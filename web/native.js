'use strict';
(function(){
 const host=document.getElementById('host');
 try{
  const fresh=location.hash.length>1;
  const raw=fresh?decodeURIComponent(location.hash.slice(1)):sessionStorage.getItem('network-speed-test-access');
  history.replaceState(null,'',location.pathname);
  if(!raw)throw Error('Open Speed Test from the router UI and press Start.');
  const access=JSON.parse(raw),back=new URL(access.back);
  if(!/^[a-f0-9]{64}$/.test(access.token)||back.hostname!==location.hostname||!['http:','https:'].includes(back.protocol)||!back.pathname.startsWith('/cgi-bin/luci/admin/network/speed_test'))throw Error('Open Speed Test from the router UI and press Start.');
  try{sessionStorage.setItem('network-speed-test-access',raw);}catch(e){}
  document.getElementById('router-link').href=back.href;
  const disclaimer=new URL(back);disclaimer.pathname='/cgi-bin/luci/admin/network/speed_test/disclaimer';disclaimer.search='';disclaimer.hash='';
  document.getElementById('disclaimer-link').href=disclaimer.href;
  for(const path of access.styles||[]){
   if(typeof path!=='string'||!/^\/luci-static\/[a-zA-Z0-9_./-]+\.css$/.test(path)||path.includes('..'))continue;
   const link=document.createElement('link');link.rel='stylesheet';link.href=path;document.head.prepend(link);
  }
  const runner=NetworkSpeedTest.mount(host,{session:access.token,endpoint:'/api',direct:true});
  // Only the handoff following the user's Start click begins a test automatically.
  if(fresh)runner.run();
 }catch(e){host.textContent=e.message;document.getElementById('router-link').href='https://'+location.hostname+'/cgi-bin/luci/admin/network/speed_test';document.getElementById('disclaimer-link').hidden=true;}
})();
