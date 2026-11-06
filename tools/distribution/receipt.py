"""Capture packaging data before publication; no publishing or write credentials."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

REPOSITORY='darkquasar/patronus'
OWNER_ID=28435534
WORKFLOW='.github/workflows/release.yml'
VERSION=r'(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)'
ASSETS=tuple('patronus-'+os_name+'-'+arch+('.exe' if os_name=='windows' else '')
             for os_name in ['darwin','linux','windows'] for arch in ['amd64','arm64'])
RECOVERY='Inspect and explicitly approve the published release baseline in the controller; do not rebuild the release to recover its receipt.'


def require(condition,message):
    if not condition:
        raise ValueError(message)


def stable(tag):
    return type(tag) is str and re.fullmatch('v'+VERSION,tag) is not None


def sha(data):
    return hashlib.sha256(data).hexdigest()


def strict_json(data):
    def unique(pairs):
        result={}
        for key,value in pairs:
            require(key not in result,'duplicate JSON key')
            result[key]=value
        return result
    value=json.loads(data,object_pairs_hook=unique,parse_float=lambda _: (_ for _ in ()).throw(ValueError('float not allowed')),
        parse_constant=lambda _: (_ for _ in ()).throw(ValueError('non-finite JSON')))
    return value


def checksums(data):
    require(type(data) is bytes and 0<len(data)<=1024*1024,'invalid checksum file')
    result={}
    for line in data.decode('utf-8').splitlines():
        match=re.fullmatch(r'([0-9a-fA-F]{64}) [ *]([^\r\n]+)',line)
        require(match is not None,'malformed checksum entry')
        digest,name=match.groups()
        require(name not in result,'duplicate checksum entry')
        result[name]=digest.lower()
    require(set(result)==set(ASSETS),'incomplete checksum set')
    return result


def capture(dist,env,source):
    ref=env.get('GITHUB_REF','')
    tag=ref.removeprefix('refs/tags/')
    if ref.startswith('refs/tags/') and not stable(tag):
        return None
    require(env.get('GITHUB_REPOSITORY')==REPOSITORY and env.get('GITHUB_EVENT_NAME')=='push'
        and ref=='refs/tags/'+tag and stable(tag),'ineligible source context')
    require(env.get('GITHUB_ACTOR_ID')==str(OWNER_ID) and env.get('GITHUB_ACTOR')=='darkquasar'
        and env.get('GITHUB_TRIGGERING_ACTOR')=='darkquasar'
        and source.owner_id('darkquasar')==OWNER_ID,'owner required')
    commit=source.commit(ref)
    require(re.fullmatch(r'[0-9a-f]{40}',commit) and env.get('GITHUB_SHA')==commit,'commit mismatch')
    run_id=int(env['GITHUB_RUN_ID']);attempt=int(env['GITHUB_RUN_ATTEMPT'])
    require(run_id>0 and attempt>0,'invalid run locator')
    workflow=source.workflow(commit)
    require(type(workflow) is bytes and workflow,'workflow observation missing')
    dist=Path(dist)
    required=[dist/name for name in (*ASSETS,'checksums.txt')]
    require(all(p.is_file() and not p.is_symlink() for p in required),'required file missing or symlink')
    checksum_bytes=(dist/'checksums.txt').read_bytes()
    hashes=checksums(checksum_bytes)
    assets={}
    for name in ASSETS:
        data=(dist/name).read_bytes()
        require(data and sha(data)==hashes[name],'release checksum mismatch')
        assets[name]={'size':len(data),'sha256':sha(data)}
    return dict(schema_version=1,repository=REPOSITORY,tag=tag,version=tag[1:],commit=commit,
        run_id=run_id,attempt=attempt,workflow_path=WORKFLOW,workflow_revision=commit,
        workflow_sha256=sha(workflow),event='push',ref=ref,actor_id=OWNER_ID,triggering_actor_id=OWNER_ID,
        checksums_sha256=sha(checksum_bytes),assets=assets)


def validate(document):
    fields='schema_version repository tag version commit run_id attempt workflow_path workflow_revision workflow_sha256 event ref actor_id triggering_actor_id checksums_sha256 assets'
    require(type(document) is dict and set(document)==set(fields.split()),'unknown or missing receipt fields')
    require(type(document['schema_version']) is int and document['schema_version']==1,'unsupported receipt')
    require(document['repository']==REPOSITORY and stable(document['tag'])
        and document['version']==document['tag'][1:] and document['ref']=='refs/tags/'+document['tag']
        and document['event']=='push' and document['workflow_path']==WORKFLOW,'receipt source mismatch')
    for field in ['actor_id','triggering_actor_id']:
        require(type(document[field]) is int and document[field]==OWNER_ID,'receipt actor mismatch')
    require(document['workflow_revision']==document['commit']
        and re.fullmatch(r'[0-9a-f]{40}',document['commit']),'receipt commit mismatch')
    for field in ['run_id','attempt']:
        require(type(document[field]) is int and document[field]>0,'invalid receipt run locator')
    for field in ['workflow_sha256','checksums_sha256']:
        require(type(document[field]) is str and re.fullmatch(r'[0-9a-f]{64}',document[field]),'invalid digest')
    require(type(document['assets']) is dict and set(document['assets'])==set(ASSETS),'receipt asset scope mismatch')
    for asset in document['assets'].values():
        require(type(asset) is dict and set(asset)=={'size','sha256'} and type(asset['size']) is int
            and asset['size']>0 and type(asset['sha256']) is str
            and re.fullmatch(r'[0-9a-f]{64}',asset['sha256']),'invalid receipt asset')
    return document


def packaging_status(tag,capture_outcome,upload_outcome):
    if not stable(tag):
        return 'skipped: tag is outside the stable package track.'
    if capture_outcome!='success' or upload_outcome!='success':
        return 'receipt-unavailable: '+RECOVERY
    return 'Receipt uploaded; automatic preparation waits for this exact release attempt to succeed.'


class Source:
    def owner_id(self,login):
        return json.loads(subprocess.check_output(['gh','api','users/'+login],timeout=30))['id']

    def commit(self,ref):
        return subprocess.check_output(['git','rev-parse','--verify',ref+'^{commit}'],timeout=30,text=True).strip()

    def workflow(self,commit):
        return subprocess.check_output(['git','show',commit+':'+WORKFLOW],timeout=30)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    sub=parser.add_subparsers(dest='command',required=True)
    capture_parser=sub.add_parser('capture')
    capture_parser.add_argument('--dist',required=True);capture_parser.add_argument('--out',required=True)
    summary=sub.add_parser('summary')
    summary.add_argument('--tag',required=True);summary.add_argument('--capture',required=True);summary.add_argument('--upload',required=True)
    args=parser.parse_args()
    if args.command=='summary':
        message=packaging_status(args.tag,args.capture,args.upload)
        print(message)
        if os.environ.get('GITHUB_STEP_SUMMARY'):
            with open(os.environ['GITHUB_STEP_SUMMARY'],'a') as stream:
                stream.write(message+'\n')
        return
    try:
        document=capture(args.dist,os.environ,Source())
        if document is not None:
            validate(document)
            path=Path(args.out);path.parent.mkdir(parents=True,exist_ok=True)
            path.write_text(json.dumps(document,sort_keys=True,separators=(',',':'))+'\n')
        if os.environ.get('GITHUB_OUTPUT'):
            with open(os.environ['GITHUB_OUTPUT'],'a') as stream:
                stream.write('eligible='+str(document is not None).lower()+'\n')
    except Exception:
        print('receipt-unavailable: '+RECOVERY,file=sys.stderr)
        raise


if __name__=='__main__':
    main()
