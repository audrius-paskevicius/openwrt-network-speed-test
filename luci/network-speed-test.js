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
  speedtest.mount(host,{session:L.env.sessionid});return host;
 },
 handleSave:null,handleSaveApply:null,handleReset:null
});
