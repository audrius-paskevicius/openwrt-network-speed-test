'use strict';
'require request';
'require baseclass';

const key='network-speed-test-disclaimer';
let acknowledgedText=null;
function accepted(text){if(acknowledgedText===text)return true;try{return window.localStorage.getItem(key)===text;}catch(e){return false;}}
function fullText(text){return E('div',{class:'nst-disclaimer-full'},text.trim().split(/\n\s*\n/).map((paragraph,i)=>E(i?'p':'h3',{},paragraph)));}
function stylesheet(){return E('link',{rel:'stylesheet',href:L.resource('network-speed-test/disclaimer.css')+'?v=@VERSION@'});}
return baseclass.extend({
 load:function(){return request.get(L.resource('network-speed-test/disclaimer.txt'),{cache:false,timeout:10000}).then(response=>{
  if(!response.ok)throw Error('Unable to load the Network Speed Test disclaimer.');
  const text=response.text();if(!text.trim())throw Error('The Network Speed Test disclaimer is empty.');return text;
 });},
 page:function(text){return E('div',{class:'cbi-map'},[stylesheet(),E('h2',{},'Network Speed Test: disclaimer'),E('section',{class:'nst-disclaimer-notice'},fullText(text))]);},
 gate:function(text,render){
  const host=E('div',{class:'cbi-map nst-disclaimer-host'},[stylesheet()]);
  function show(){return Promise.resolve().then(render).then(page=>{host.replaceChildren(host.firstChild,page);return host;});}
  if(accepted(text))return show();
  const error=E('p',{role:'alert'});
  const button=E('button',{type:'button',class:'cbi-button cbi-button-action',click:function(){
   button.disabled=true;error.textContent='';acknowledgedText=text;try{window.localStorage.setItem(key,text);}catch(e){}
   return show().catch(e=>{button.disabled=false;error.textContent='Unable to load this page: '+e.message;});
  }},'I understand — continue');
  host.appendChild(E('section',{class:'nst-disclaimer-notice'},[fullText(text),E('p',{},'Acknowledgement is remembered in this browser for this router while its site data is retained, including after a computer restart. It does not start a test or change router settings.'),button,error]));
  return Promise.resolve(host);
 }
});
