const fs=require('node:fs');
for(const path of ['web/test.js','web/native.js','luci/network-speed-test.js','luci/disclaimer.js','luci/network-speed-test-disclaimer.js']){new Function(fs.readFileSync(path,'utf8'));console.log('Syntax OK: '+path);}
for(const path of ['files/menu.json','files/acl.json'])JSON.parse(fs.readFileSync(path,'utf8'));
