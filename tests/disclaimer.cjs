const fs=require('node:fs'),assert=require('node:assert/strict');
const source=fs.readFileSync('luci/disclaimer.js','utf8'),text=fs.readFileSync('DISCLAIMER.txt','utf8');
function E(tag,attrs={},children=[]){return {tag,attrs,children:Array.isArray(children)?children:[children],get firstChild(){return this.children[0];},appendChild(n){this.children.push(n);},replaceChildren(...nodes){this.children=nodes;}};}
function find(n,tag){if(n.tag===tag)return n;for(const c of n.children||[]){if(c&&typeof c==='object'){const found=find(c,tag);if(found)return found;}}}
function fixture(storage,response={ok:true,text:()=>text}){return new Function('request','baseclass','window','E','L',source)({get:async()=>response},{extend:x=>x},{localStorage:storage},E,{resource:x=>x});}
(async()=>{
 let saved=null,renders=0;
 const storage={getItem:()=>saved,setItem:(key,value)=>{assert.equal(key,'network-speed-test-disclaimer');saved=value;}};
 const render=()=>{renders++;return E('main',{},'Test');},api=fixture(storage);
 assert.equal(await api.load(),text);
 const page=await api.gate(text,render);assert.equal(renders,0);assert.ok(JSON.stringify(page).includes(text.trim().split('\n\n')[2]));
 await find(page,'button').attrs.click();assert.equal(renders,1);assert.equal(saved,text);
 await fixture(storage).gate(text,render);assert.equal(renders,2,'acceptance survives a fresh module');
 const changed=await fixture(storage).gate(text+'\nChanged text.',render);assert.equal(renders,2);assert.ok(find(changed,'button'));
 assert.ok(JSON.stringify(api.page(text)).includes('Network Speed Test: disclaimer'));
 const blocked=fixture({getItem(){throw Error('blocked');},setItem(){throw Error('blocked');}}),gate=await blocked.gate(text,render);await find(gate,'button').attrs.click();await blocked.gate(text,render);assert.equal(renders,4);
 await assert.rejects(fixture(storage,{ok:false}).load(),/Unable to load/);await assert.rejects(fixture(storage,{ok:true,text:()=>''}).load(),/empty/);
 const readme=fs.readFileSync('README.md','utf8');for(const p of text.trim().split('\n\n'))assert.ok(readme.includes(p));
 console.log('PASS: canonical disclaimer, first-use gate, exact-text persistence, changed wording, blocked storage and load failures');
})().catch(e=>{console.error(e);process.exitCode=1});
