'use strict';
'require view';
'require network-speed-test.test as speedtest';
'require network-speed-test.disclaimer as warning';

return view.extend({
 load:function(){return warning.load();},
 render:function(text){return warning.gate(text,()=>this.renderPage());},
 renderPage:function(){
  const host=E('div',{class:'cbi-map'},[
   E('link',{rel:'stylesheet',href:L.resource('network-speed-test/style.css')+'?v=@VERSION@'})
  ]);
  speedtest.mount(host,{session:L.env.sessionid,launch:async function(){
   const response=await fetch('/cgi-bin/network-speed-test?op=prepare',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{Authorization:'Bearer '+L.env.sessionid}});
   const data=await response.json();
   if(!response.ok)throw Error(data.error||'Could not prepare the local test.');
   const styles=[...document.querySelectorAll('link[rel=stylesheet]')].map(link=>new URL(link.href)).filter(url=>url.origin===location.origin&&url.pathname.startsWith('/luci-static/')).map(url=>url.pathname);
   const target=new URL(location.href);target.protocol='http:';target.port='18080';target.pathname='/test/';target.search='';
   target.hash=encodeURIComponent(JSON.stringify({token:data.token,back:location.href,styles}));
   location.assign(target.href);
  }});return host;
 },
 handleSave:null,handleSaveApply:null,handleReset:null
});
