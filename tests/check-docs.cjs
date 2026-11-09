const fs=require('node:fs'),path=require('node:path'),assert=require('node:assert/strict');
const read=p=>fs.readFileSync(p,'utf8').replace(/\r\n/g,'\n');
const version=read('VERSION').trim(),readme=read('README.md');
assert.match(version,/^\d+\.\d+\.\d+-r\d+$/);
const notice=readme.split('<!-- project-disclaimer:start -->')[1].split('<!-- project-disclaimer:end -->')[0].trim().split('\n').slice(1).map(s=>s.replace(/^> ?/,'')).join('\n').trim();
assert.equal(notice,read('DISCLAIMER.txt').trim(),'README disclaimer must match canonical text');
assert.ok(read('CHANGELOG.md').includes('## '+version+' '),'changelog must cover current package');
for(const match of readme.matchAll(/luci-app-network-speed-test-(\d+\.\d+\.\d+-r\d+)\.apk/g))assert.equal(match[1],version,'install example version');
const docs=['README.md','CHANGELOG.md','SECURITY.md','CONTRIBUTING.md',...fs.readdirSync('docs').filter(f=>f.endsWith('.md')).map(f=>'docs/'+f)];
for(const file of docs){
 for(const [,target] of read(file).matchAll(/\]\(([^\s)]+)\)/g)){
  if(/^(https?:|#)/.test(target))continue;
  const dest=decodeURIComponent(target.split('#')[0]);
  assert.ok(fs.existsSync(path.resolve(path.dirname(file),dest)),file+': missing '+dest);
 }
}
assert.ok(!readme.includes('0.2.1-r2'),'unpublished intermediate version');
console.log('PASS: current version, canonical disclaimer and local documentation links');
