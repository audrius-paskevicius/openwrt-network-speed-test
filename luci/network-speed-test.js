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
  speedtest.mount(host,{session:L.env.sessionid,prepare:async function(){
   const response=await fetch('/cgi-bin/network-speed-test?op=prepare',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{Authorization:'Bearer '+L.env.sessionid}});
   const data=await response.json();
   if(!response.ok)throw Error(data.error||'Could not prepare the local test.');
   const target=new URL(location.href);target.protocol='http:';target.port='18080';target.pathname='/api';target.search='';target.hash='';
   return {session:data.token,endpoint:target.href,direct:true};
  }});return host;
 },
 handleSave:null,handleSaveApply:null,handleReset:null
});
