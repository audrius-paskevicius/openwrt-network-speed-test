'use strict';
'require view';
'require network-speed-test.disclaimer as warning';

return view.extend({
 load:function(){return warning.load();},
 render:function(text){return warning.gate(text,()=>warning.page(text));},
 handleSave:null,handleSaveApply:null,handleReset:null
});
