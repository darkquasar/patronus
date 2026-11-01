"""Offline public behavior checks. All release bytes and API responses are invented."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location('receipt',ROOT/'scripts/distribution/receipt.py')
receipt=importlib.util.module_from_spec(SPEC);SPEC.loader.exec_module(receipt)


class Source:
    def owner_id(self, login):
        return 28435534

    def commit(self, ref):
        return 'a'*40

    def workflow(self, commit):
        return b'name: release\non: {push: {tags: ["v*"]}}\n'


def environment():
    return dict(GITHUB_REPOSITORY='darkquasar/patronus',GITHUB_EVENT_NAME='push',
        GITHUB_REF='refs/tags/v2.8.0',GITHUB_SHA='a'*40,GITHUB_RUN_ID='50',GITHUB_RUN_ATTEMPT='1',
        GITHUB_ACTOR_ID='28435534',GITHUB_ACTOR='darkquasar',GITHUB_TRIGGERING_ACTOR='darkquasar')


class ReceiptTests(unittest.TestCase):
    def test_nonowner_cannot_capture_an_authorized_receipt(self):
        with self.assertRaises(ValueError):
            receipt.capture(Path('/unused'),{**environment(),'GITHUB_ACTOR_ID':'123'},Source())

    def test_capture_binds_all_six_bytes_and_audits_workflow(self):
        with tempfile.TemporaryDirectory() as folder:
            dist=Path(folder)
            for name in ['patronus-'+os+'-'+arch+('.exe' if os=='windows' else '')
                         for os in ['darwin','linux','windows'] for arch in ['amd64','arm64']]:
                (dist/name).write_bytes(('invented '+name).encode())
            (dist/'checksums.txt').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n'
                for p in sorted(dist.iterdir())))
            captured=receipt.capture(dist,environment(),Source())
            self.assertEqual(captured['version'],'2.8.0')
            self.assertEqual(len(captured['assets']),6)
            self.assertEqual(captured['workflow_sha256'],hashlib.sha256(Source().workflow('a'*40)).hexdigest())
            name=next(iter(captured['assets']))
            (dist/name).write_bytes(b'changed bytes')
            with self.assertRaises(ValueError):
                receipt.capture(dist,environment(),Source())

    def test_ineligible_tags_skip_and_untrusted_capture_contexts_deny(self):
        for tag in ['v2.8.0-rc.1','v02.8.0','v2.8.0+build']:
            self.assertIsNone(receipt.capture(Path('/unused'),{**environment(),'GITHUB_REF':'refs/tags/'+tag},Source()))
        for key,value in [('GITHUB_REPOSITORY','fork/patronus'),('GITHUB_EVENT_NAME','pull_request'),
            ('GITHUB_TRIGGERING_ACTOR','other'),('GITHUB_SHA','b'*40),('GITHUB_RUN_ATTEMPT','0')]:
            with self.subTest(key=key),self.assertRaises(ValueError):
                receipt.capture(Path('/unused'),{**environment(),key:value},Source())

DISPATCH_SPEC=importlib.util.spec_from_file_location('dispatch',ROOT/'scripts/distribution/dispatch.py')
dispatch=importlib.util.module_from_spec(DISPATCH_SPEC);DISPATCH_SPEC.loader.exec_module(dispatch)


class DispatchTests(unittest.TestCase):
    def test_fork_event_never_accesses_dispatch_credentials(self):
        minted=[]
        with self.assertRaises(ValueError):
            dispatch.handle({'workflow_run':{'id':50,'run_attempt':1}},
                {'GITHUB_REPOSITORY':'fork/patronus'},None,lambda: minted.append(True))
        self.assertEqual(minted,[])


class Service:
    """External GitHub HTTP boundary with actual API-shaped response objects."""
    def __init__(self):
        import io,zipfile,base64
        with tempfile.TemporaryDirectory() as folder:
            dist=Path(folder)
            for name in receipt.ASSETS:
                (dist/name).write_bytes(('invented '+name).encode())
            checks=''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n' for p in sorted(dist.iterdir()))
            (dist/'checksums.txt').write_text(checks)
            self.receipt=receipt.capture(dist,environment(),Source())
        repo={'id':123,'full_name':receipt.REPOSITORY,'owner':{'id':28435534},'default_branch':'main'}
        self.repo=repo
        common=dict(repository=repo,head_repository=repo,actor={'id':28435534},triggering_actor={'id':28435534})
        self.receiver=dict(common,id=40,run_attempt=1,path='.github/workflows/package-distribution.yml',
            event='workflow_run',head_branch='main',head_sha='c'*40)
        self.run=dict(common,id=50,run_attempt=1,path=receipt.WORKFLOW,event='push',head_branch='v2.8.0',
            head_sha='a'*40,status='completed',conclusion='success')
        self.artifact={'id':700,'name':'distribution-receipt-50-1','expired':False,
            'workflow_run':{'id':50,'head_sha':'a'*40}}
        self.release={'id':600,'tag_name':'v2.8.0','draft':False,'prerelease':False}
        self.assets=[{'name':name,'size':data['size'],'digest':'sha256:'+data['sha256']}
                     for name,data in self.receipt['assets'].items()]
        self.assets.append({'name':'checksums.txt','size':len(checks.encode()),'digest':'sha256:'+self.receipt['checksums_sha256']})
        self.tip='c'*40;self.commit='a'*40;self.ancestry='ahead';self.missing_artifact=False
        self.calls=[]

    def __call__(self,method,path,body=None,token=None):
        import base64,io,zipfile
        self.calls.append((method,path,body,token))
        if method=='POST':
            return b''
        prefix='repos/darkquasar/patronus'
        if path==prefix: result=self.repo
        elif '/actions/runs/40/attempts/1' in path: result=self.receiver
        elif '/actions/runs/50/attempts/1' in path: result=self.run
        elif '/git/ref/heads/main' in path: result={'object':{'type':'commit','sha':self.tip}}
        elif '/git/ref/tags/v2.8.0' in path: result={'object':{'type':'commit','sha':self.commit}}
        elif '/compare/' in path: result={'status':self.ancestry}
        elif '/git/trees/' in path: result={'truncated':False,'tree':[{'path':receipt.WORKFLOW,'mode':'100644','type':'blob','sha':'e'*40}]}
        elif '/git/blobs/' in path: result={'encoding':'base64','content':base64.b64encode(Source().workflow('a'*40)).decode()}
        elif '/actions/runs/50/artifacts' in path:
            stream=io.BytesIO()
            with zipfile.ZipFile(stream,'w') as archive:
                archive.writestr('receipt.json',json.dumps(self.receipt))
            self.zip=stream.getvalue()
            self.artifact['digest']='sha256:'+hashlib.sha256(self.zip).hexdigest()
            result={'total_count':0 if self.missing_artifact else 1,'artifacts':[] if self.missing_artifact else [self.artifact]}
        elif '/actions/artifacts/700/zip' in path: return self.zip
        elif '/releases/tags/' in path: result=self.release
        elif '/releases/600/assets' in path: result=self.assets
        else: raise AssertionError('unexpected GitHub request '+path)
        return json.dumps(result).encode()


def receiver_environment():
    return dict(GITHUB_REPOSITORY=receipt.REPOSITORY,GITHUB_EVENT_NAME='workflow_run',
        GITHUB_REF='refs/heads/main',GITHUB_SHA='c'*40,GITHUB_RUN_ID='40',GITHUB_RUN_ATTEMPT='1')


def completion():
    return {'repository':{'full_name':receipt.REPOSITORY},'workflow_run':{'id':50,'run_attempt':1}}


class CompletedReleaseTests(unittest.TestCase):
    def deliver(self,service=None,event=None,env=None):
        service=service or Service();minted=[]
        result=dispatch.handle(event or completion(),env or receiver_environment(),service,
            lambda: minted.append('requested') or 'invented-dispatch-token')
        return result,minted,service

    def test_valid_completed_release_dispatches_fixed_controller_and_ref(self):
        result,minted,service=self.deliver()
        self.assertEqual(result['state'],'dispatched')
        self.assertEqual(minted,['requested'])
        writes=[call for call in service.calls if call[0]=='POST']
        self.assertEqual(len(writes),1)
        self.assertEqual(writes[0][1],'repos/darkquasar/homebrew-tap/actions/workflows/distribute.yml/dispatches')
        self.assertEqual(writes[0][2],{'ref':'distribution-control','inputs':{'mode':'prepare','app':'patronus','run_id':'50','attempt':'1'}})
        self.assertEqual(writes[0][3],'invented-dispatch-token')

    def test_failed_wrong_or_untrusted_runs_never_mint(self):
        for target,field,value in [('run','conclusion','failure'),('run','event','pull_request'),
            ('run','path','.github/workflows/publish-catalog.yml'),('run','run_attempt',2),
            ('run','actor',{'id':123}),('run','triggering_actor',{'id':123}),
            ('run','head_repository',{'id':999,'full_name':'fork/patronus'}),
            ('receiver','triggering_actor',{'id':123}),('receiver','head_sha','f'*40),
            ('receiver','path','.github/workflows/untrusted.yml')]:
            service=Service();getattr(service,target)[field]=value;minted=[]
            with self.subTest(target=target,field=field),self.assertRaises(ValueError):
                dispatch.handle(completion(),receiver_environment(),service,lambda:minted.append(True))
            self.assertEqual(minted,[])


class CommandTests(unittest.TestCase):
    def test_handler_cli_is_importable_from_checkout(self):
        import subprocess,sys
        result=subprocess.run([sys.executable,'-B','scripts/distribution/dispatch.py','--help'],
            cwd=ROOT,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        self.assertEqual(result.returncode,0,result.stderr.decode())


class FailurePolicyTests(unittest.TestCase):
    def test_missing_or_corrupt_receipt_and_failed_publication_never_mint(self):
        for change in ['missing','upload-failed','generation-failed','failed-publication','checksum','asset','workflow','stale']:
            service=Service();minted=[]
            if change in ['missing','upload-failed','generation-failed']:
                service.missing_artifact=True
            elif change=='failed-publication': service.run['conclusion']='failure'
            elif change=='checksum': service.assets[-1]['digest']='sha256:'+'0'*64
            elif change=='asset': service.assets[0]['digest']='sha256:'+'0'*64
            elif change=='workflow': service.receipt['workflow_sha256']='0'*64
            else: service.tip='f'*40
            with self.subTest(change=change),self.assertRaises(ValueError):
                dispatch.handle(completion(),receiver_environment(),service,lambda:minted.append(True))
            self.assertEqual(minted,[])
        for capture,upload in [('failure','skipped'),('success','failure')]:
            self.assertIn('receipt-unavailable',receipt.packaging_status('v2.8.0',capture,upload))
            self.assertIn('do not rebuild',receipt.packaging_status('v2.8.0',capture,upload))

    def test_suffix_skips_and_owner_dispatch_recovery_reuses_original_receipt(self):
        service=Service();service.run['head_branch']='v2.8.0-docs.1';minted=[]
        self.assertEqual(dispatch.handle(completion(),receiver_environment(),service,lambda:minted.append(True))['state'],'skipped')
        self.assertEqual(minted,[])
        service=Service();service.receiver['event']='workflow_dispatch'
        env={**receiver_environment(),'GITHUB_EVENT_NAME':'workflow_dispatch'}
        event={'repository':{'full_name':receipt.REPOSITORY},'inputs':{'run_id':'50','attempt':'1'}}
        result=dispatch.handle(event,env,service,lambda:'invented-token')
        self.assertEqual(result['request']['inputs']['run_id'],'50')
        self.assertTrue(all(call[1]==dispatch.DESTINATION for call in service.calls if call[0]!='GET'))
        service.receiver['triggering_actor']={'id':123}
        with self.assertRaises(ValueError):
            dispatch.handle(event,env,service,lambda:minted.append(True))
        self.assertEqual(minted,[])

    def test_content_only_event_and_cross_app_event_never_mint(self):
        for event,env in [({'repository':{'full_name':'other/lumen'}},receiver_environment()),
                (completion(),{**receiver_environment(),'GITHUB_EVENT_NAME':'repository_dispatch'})]:
            minted=[]
            with self.assertRaises(ValueError):
                dispatch.handle(event,env,Service(),lambda:minted.append(True))
            self.assertEqual(minted,[])

    def test_wrong_artifact_attempt_expiry_receipt_and_release_flags_deny_before_token(self):
        for change in ['attempt','expired','artifact-name','draft','prerelease','duplicate-json','symlink']:
            service=Service();minted=[]
            if change=='attempt': service.receipt['attempt']=2
            elif change=='expired': service.artifact['expired']=True
            elif change=='artifact-name': service.artifact['name']='distribution-receipt-50-2'
            elif change in ['draft','prerelease']: service.release[change]=True
            else:
                original=service.__call__
                def api(method,path,body=None,token=None,obj=service):
                    import io,zipfile
                    result=original(method,path,body,token)
                    if '/actions/runs/50/artifacts' in path:
                        stream=io.BytesIO()
                        with zipfile.ZipFile(stream,'w') as archive:
                            info=zipfile.ZipInfo('receipt.json')
                            content=json.dumps(obj.receipt)
                            if change=='symlink': info.external_attr=0o120777<<16
                            else: content=content[:-1]+',"schema_version":1}'
                            archive.writestr(info,content)
                        obj.zip=stream.getvalue()
                        obj.artifact['digest']='sha256:'+receipt.sha(obj.zip)
                        return json.dumps({'total_count':1,'artifacts':[obj.artifact]}).encode()
                    return result
                service=api
            with self.subTest(change=change),self.assertRaises(ValueError):
                dispatch.handle(completion(),receiver_environment(),service,lambda:minted.append(True))
            self.assertEqual(minted,[])
