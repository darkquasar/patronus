"""Admit a completed release before obtaining a tap-only dispatch credential."""
import argparse
import base64
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import zipfile

# Keep script and unittest entry points on the same implementation.
if __package__ in (None, ""):
    sys.path.insert(0,str(Path(__file__).resolve().parents[2]))
from scripts.distribution import receipt

DESTINATION='repos/darkquasar/homebrew-tap/actions/workflows/distribute.yml/dispatches'
HANDLER='.github/workflows/package-distribution.yml'
PREFIX='repos/'+receipt.REPOSITORY
SHA=r'[0-9a-f]{40}'


def get(api,path):
    data=api('GET',path)
    receipt.require(type(data) is bytes and len(data)<=8*1024*1024,'API response unavailable or too large')
    return receipt.strict_json(data)


def listing(api,path,key=None):
    items=[]
    for page in range(1,21):
        value=get(api,path+f'?per_page=100&page={page}')
        batch=value[key] if key else value
        receipt.require(type(batch) is list,'invalid paginated response')
        items.extend(batch)
        if len(batch)<100:
            if key:
                receipt.require(len(items)==value['total_count'],'incomplete artifact listing')
            return items
    raise ValueError('pagination bound exceeded')


def run(api,run_id,attempt):
    receipt.require(type(run_id) is int and run_id>0 and type(attempt) is int and attempt>0,'invalid run reference')
    data=get(api,f'{PREFIX}/actions/runs/{run_id}/attempts/{attempt}')
    receipt.require(data.get('id')==run_id and data.get('run_attempt')==attempt,'run/attempt mismatch')
    return data


def ref_commit(api,ref):
    obj=get(api,f'{PREFIX}/git/ref/{ref}')['object']
    seen=set()
    for _ in range(16):
        receipt.require(type(obj) is dict and re.fullmatch(SHA,obj.get('sha','')),'invalid Git ref')
        if obj.get('type')=='commit':
            return obj['sha']
        receipt.require(obj.get('type')=='tag' and obj['sha'] not in seen,'invalid/cyclic Git tag')
        seen.add(obj['sha'])
        obj=get(api,f"{PREFIX}/git/tags/{obj['sha']}")['object']
    raise ValueError('tag dereference bound exceeded')


def owner_run(data,repo):
    receipt.require(data.get('repository',{}).get('full_name')==receipt.REPOSITORY
        and data['repository'].get('id')==repo['id']
        and data.get('head_repository',{}).get('full_name')==receipt.REPOSITORY
        and data['head_repository'].get('id')==repo['id'],'fork or other repository run')
    for field in ['actor','triggering_actor']:
        actor=data.get(field,{}).get('id')
        receipt.require(type(actor) is int and actor==receipt.OWNER_ID,'non-owner run/rerun')


def verified_request(event,env,api):
    receipt.require(env.get('GITHUB_REPOSITORY')==receipt.REPOSITORY,'wrong receiver repository')
    repo=get(api,PREFIX)
    receipt.require(repo.get('full_name')==receipt.REPOSITORY and repo.get('owner',{}).get('id')==receipt.OWNER_ID
        and repo.get('default_branch')=='main','unapproved producer authority/default branch')
    sha=env.get('GITHUB_SHA','')
    receipt.require(env.get('GITHUB_REF')=='refs/heads/main' and re.fullmatch(SHA,sha)
        and ref_commit(api,'heads/main')==sha,'receiver must use fresh main code')
    receiver=run(api,int(env['GITHUB_RUN_ID']),int(env['GITHUB_RUN_ATTEMPT']))
    owner_run(receiver,repo)
    kind=env.get('GITHUB_EVENT_NAME')
    receipt.require(kind in {'workflow_run','workflow_dispatch'} and receiver.get('event')==kind
        and receiver.get('path')==HANDLER and receiver.get('head_branch')=='main'
        and receiver.get('head_sha')==sha,'receiver workflow mismatch')
    receipt.require(event.get('repository',{}).get('full_name')==receipt.REPOSITORY,'cross-app event')
    if kind=='workflow_run':
        locator=event.get('workflow_run',{})
        source_id,attempt=locator.get('id'),locator.get('run_attempt')
    else:
        inputs=event.get('inputs',{})
        receipt.require(set(inputs)=={'run_id','attempt'},'unexpected recovery inputs')
        source_id,attempt=int(inputs['run_id']),int(inputs['attempt'])
    source=run(api,source_id,attempt)
    owner_run(source,repo)
    receipt.require(source.get('path')==receipt.WORKFLOW and source.get('event')=='push'
        and source.get('status')=='completed' and source.get('conclusion')=='success','source release has not succeeded')
    tag=source.get('head_branch')
    if not receipt.stable(tag):
        return None
    commit=ref_commit(api,'tags/'+tag)
    receipt.require(commit==source.get('head_sha'),'release tag/run commit conflict')
    ancestry=get(api,f'{PREFIX}/compare/{commit}...{sha}')
    receipt.require(ancestry.get('status') in {'ahead','identical'},'release outside main history')
    artifacts=listing(api,f'{PREFIX}/actions/runs/{source_id}/artifacts','artifacts')
    name=f'distribution-receipt-{source_id}-{attempt}'
    matches=[a for a in artifacts if a.get('name')==name]
    receipt.require(len(matches)==1,'receipt-unavailable: '+receipt.RECOVERY)
    artifact=matches[0]
    receipt.require(type(artifact.get('id')) is int and artifact['id']>0 and artifact.get('expired') is False
        and artifact.get('workflow_run',{}).get('id')==source_id
        and artifact['workflow_run'].get('head_sha')==commit,'artifact provenance mismatch')
    digest=artifact.get('digest')
    receipt.require(type(digest) is str and re.fullmatch(r'sha256:[0-9a-f]{64}',digest),'artifact API digest unavailable')
    content=api('GET',f"{PREFIX}/actions/artifacts/{artifact['id']}/zip")
    receipt.require(type(content) is bytes and len(content)<=1024*1024 and 'sha256:'+receipt.sha(content)==digest,
        'artifact byte digest mismatch')
    with zipfile.ZipFile(io.BytesIO(content)) as archive:
        receipt.require(archive.namelist()==['receipt.json'],'unexpected receipt archive members')
        info=archive.getinfo('receipt.json')
        receipt.require(info.file_size<=1024*1024 and (info.external_attr>>16)&0o170000!=0o120000,'unsafe receipt member')
        document=receipt.validate(receipt.strict_json(archive.read(info)))
    expected=dict(run_id=source_id,attempt=attempt,commit=commit,workflow_revision=commit,tag=tag)
    receipt.require(all(document[k]==v for k,v in expected.items()),'receipt/run identity mismatch')
    tree=get(api,f'{PREFIX}/git/trees/{commit}?recursive=1')
    receipt.require(tree.get('truncated') is False,'workflow tree listing incomplete')
    entries=[entry for entry in tree['tree'] if entry.get('path')==receipt.WORKFLOW]
    receipt.require(len(entries)==1 and entries[0].get('mode') in {'100644','100755'}
        and entries[0].get('type')=='blob' and re.fullmatch(SHA,entries[0].get('sha','')),'unsafe workflow entry')
    blob=get(api,f"{PREFIX}/git/blobs/{entries[0]['sha']}")
    receipt.require(blob.get('encoding')=='base64','workflow blob unreadable')
    workflow=base64.b64decode(blob['content'])
    receipt.require(receipt.sha(workflow)==document['workflow_sha256'],'workflow audit mismatch')
    release=get(api,f'{PREFIX}/releases/tags/{tag}')
    receipt.require(release.get('tag_name')==tag and release.get('draft') is False
        and release.get('prerelease') is False and type(release.get('id')) is int,'not a published stable release')
    assets=listing(api,f"{PREFIX}/releases/{release['id']}/assets")
    by_name={a['name']:a for a in assets}
    receipt.require(len(by_name)==len(assets),'duplicate published assets')
    for name,data in document['assets'].items():
        actual=by_name.get(name,{})
        receipt.require(type(actual.get('size')) is int and actual['size']==data['size']
            and actual.get('digest')=='sha256:'+data['sha256'],'published asset/receipt conflict')
    receipt.require(by_name.get('checksums.txt',{}).get('digest')=='sha256:'+document['checksums_sha256'],
        'published checksum/receipt conflict')
    return {'ref':'distribution-control','inputs':{'mode':'prepare','app':'patronus',
        'run_id':str(source_id),'attempt':str(attempt)}}


def handle(event,env,github,mint):
    request=verified_request(event,env,github)
    if request is None:
        return {'state':'skipped','reason':'ineligible stable tag'}
    token=mint()
    receipt.require(type(token) is str and bool(token),'dispatch credential unavailable')
    github('POST',DESTINATION,request,token)
    return {'state':'dispatched','request':request}


def transport(method,path,body=None,token=None):
    command=['gh','api','--method',method,path]
    if body is not None:
        command+=['--input','-']
    process_env=dict(os.environ)
    if token is not None:
        process_env['GH_TOKEN']=token
    for _ in range(3 if method=='GET' else 1):
        response=subprocess.run(command,input=json.dumps(body).encode() if body is not None else None,
            stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,env=process_env,timeout=30)
        if response.returncode==0:
            return response.stdout
    raise RuntimeError('GitHub read unavailable' if method=='GET' else 'Dispatch result unknown; inspect receiver runs before owner retry')


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command',choices=['verify','send'])
    args=parser.parse_args()
    event=receipt.strict_json(Path(os.environ['GITHUB_EVENT_PATH']).read_bytes())
    try:
        if args.command=='verify':
            request=verified_request(event,os.environ,transport)
            if os.environ.get('GITHUB_OUTPUT'):
                with open(os.environ['GITHUB_OUTPUT'],'a') as stream:
                    stream.write('authorized='+str(request is not None).lower()+'\n')
            result={'state':'verified' if request else 'skipped'}
        else:
            result=handle(event,os.environ,transport,lambda:os.environ['DISPATCH_TOKEN'])
        message=json.dumps(result,sort_keys=True)
        print(message)
    except Exception:
        message='package-distribution failed: receipt/source/admission unavailable or dispatch state unknown. '+receipt.RECOVERY
        print(message,file=sys.stderr)
        raise
    finally:
        if os.environ.get('GITHUB_STEP_SUMMARY'):
            with open(os.environ['GITHUB_STEP_SUMMARY'],'a') as stream:
                stream.write(message+'\n')


if __name__=='__main__':
    # Direct file execution has scripts/distribution on sys.path; use the checkout root.
    sys.path.insert(0,str(Path(__file__).resolve().parents[2]))
    main()
