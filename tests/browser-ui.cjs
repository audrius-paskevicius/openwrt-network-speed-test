const fs=require('node:fs'),assert=require('node:assert/strict');
const endpoint='http://127.0.0.1:'+(process.env.CHROME_PORT||9223);
async function connect(){
 const tab=await(await fetch(endpoint+'/json/new?about:blank',{method:'PUT'})).json(),ws=new WebSocket(tab.webSocketDebuggerUrl);
 await new Promise((resolve,reject)=>{ws.onopen=resolve;ws.onerror=reject;});let seq=0;const pending=new Map();
 ws.onmessage=event=>{const m=JSON.parse(event.data),p=pending.get(m.id);if(p){pending.delete(m.id);clearTimeout(p.timer);m.error?p.reject(Error(m.error.message)):p.resolve(m.result);}};
 const call=(method,params={})=>new Promise((resolve,reject)=>{const id=++seq,timer=setTimeout(()=>{pending.delete(id);reject(Error('Timeout: '+method));},15000);pending.set(id,{resolve,reject,timer});ws.send(JSON.stringify({id,method,params}));});
 const run=async expression=>{const r=await call('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value;};
 return {call,run,close:async()=>{ws.close();await fetch(endpoint+'/json/close/'+tab.id);}};
}
(async()=>{
 const page=await connect();try{
  await page.call('Page.setDocumentContent',{frameId:(await page.call('Page.getFrameTree')).frameTree.frame.id,html:'<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"></head><body><div id="host"></div></body></html>'});
  await page.run(`document.head.append(Object.assign(document.createElement('style'),{textContent:${JSON.stringify(fs.readFileSync('web/style.css','utf8'))}}));window.api=new Function('baseclass',${JSON.stringify(fs.readFileSync('web/test.js','utf8'))})({extend:x=>x});window.requests=[];window.mode='normal';window.incomplete=false;window.fetch=async(url,options)=>{
   const op=new URL(url,'http://fixture.test').searchParams.get('op');requests.push({op,method:options.method,authorization:options.headers.Authorization,bytes:options.body?.byteLength||0,measurement:op==='finish'&&options.body?JSON.parse(options.body):null});
   const json=(data,status=200)=>Promise.resolve(new Response(JSON.stringify(data),{status,headers:{'Content-Type':'application/json'}}));
   if(mode==='denied')return json({error:'Your router login has expired.'},403);
   if(mode==='busy'&&op==='start')return json({error:'Another test is running.'},409);
   if(op==='connection'||op==='start')return json({connection:{radio:mode==='roamed'?'radio2':'radio0',band:'5 GHz',channel:mode==='roamed'?52:36}});
   if(op==='ping')return json({ok:true});
   if(op==='finish')return json({connection_changed:mode==='changed',router_cpu_busy:mode==='changed'?98:20});
   if(op==='upload'){await new Promise(resolve=>setTimeout(resolve,5));return json({bytes:options.body.byteLength-(incomplete?1:0)});}
   if(op==='download'){
    const stream=new ReadableStream({start(c){this.timer=setInterval(()=>c.enqueue(new Uint8Array(65536)),2);options.signal.addEventListener('abort',()=>{clearInterval(this.timer);try{c.error(new DOMException('Aborted','AbortError'));}catch(e){}});},cancel(){clearInterval(this.timer);}});return new Response(stream,{headers:{'Content-Type':'application/octet-stream'}});
   }throw Error('Unexpected endpoint');
  };window.runner=api.mount(document.getElementById('host'),{session:'synthetic-session',direct:true,timing:{warmupMs:10,minMs:60,maxMs:100,timeoutMs:2000}});`);
  assert.equal(await page.run('document.querySelectorAll("button").length'),1);
  assert.equal(await page.run('document.querySelector("button").textContent'),'Start');
  assert.equal(await page.run('requests.filter(r=>r.op!=="connection").length'),0,'no automatic tests');
  await page.run('runner.refreshConnection()');
  assert.match(await page.run('document.querySelector(".nst-connection").textContent'),/radio0.*5 GHz.*Channel 36/);
  await page.run('mode="roamed";runner.refreshConnection()');
  assert.match(await page.run('document.querySelector(".nst-connection").textContent'),/radio2.*Channel 52/);
  await page.run('mode="normal"');
  assert.deepEqual(await page.run('[...document.querySelectorAll("[data-result]")].map(x=>x.textContent)'),['—','—','—']);
  await page.run('runner.run()');
  assert.equal(await page.run('document.querySelector(".nst-status").textContent'),'Complete.');
  assert.deepEqual(await page.run('[...document.querySelectorAll("[data-result]")].map(x=>Number.isFinite(Number(x.textContent)))'),[true,true,true]);
  assert.equal(await page.run('requests.every(r=>r.method==="POST"&&r.authorization==="Bearer synthetic-session")'),true);
  assert.equal(await page.run('requests.filter(r=>r.op==="start").length'),1);
  assert.equal(await page.run('requests.find(r=>r.measurement)?.measurement.download_bytes>0 && requests.find(r=>r.measurement)?.measurement.upload_bytes>0'),true);
  assert.equal(await page.run('(requests.find(r=>r.measurement).measurement.download_bytes*8/requests.find(r=>r.measurement).measurement.download_ms/1000).toFixed(1)===document.querySelector("[data-result=download]").textContent'),true);
  for(const width of [320,390,1280]){
   await page.call('Emulation.setDeviceMetricsOverride',{width,height:844,deviceScaleFactor:1,mobile:width<650});
   assert.equal(await page.run('document.documentElement.scrollWidth<=innerWidth'),true,'layout '+width);
   if(process.env.UI_SCREENSHOTS){fs.mkdirSync('.private',{recursive:true});fs.writeFileSync('.private/speed-test-'+width+'.png',Buffer.from((await page.call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true})).data,'base64'));}
  }
  await page.run('mode="changed";runner.run()');assert.match(await page.run('document.querySelector(".nst-warning").textContent'),/radio or channel changed/);assert.match(await page.run('document.querySelector(".nst-warning").textContent'),/router was busy/);
  await page.run('mode="denied";runner.run()');assert.match(await page.run('document.querySelector(".nst-status").textContent'),/login has expired/);assert.equal(await page.run('document.querySelector("button").disabled'),false);
  await page.run('mode="busy";runner.run()');assert.match(await page.run('document.querySelector(".nst-status").textContent'),/Another test/);
  await page.run('mode="normal";incomplete=true;runner.run()');assert.match(await page.run('document.querySelector(".nst-status").textContent'),/complete upload/);assert.deepEqual(await page.run('[...document.querySelectorAll("[data-result]")].map(x=>x.textContent)'),['—','—','—']);
  await page.run('incomplete=false;window.running=runner.run();runner.run();setTimeout(()=>{Object.defineProperty(document,"hidden",{configurable:true,value:true});document.dispatchEvent(new Event("visibilitychange"));},25);running');
  assert.match(await page.run('document.querySelector(".nst-status").textContent'),/interrupted/);
  assert.equal(await page.run('document.querySelector("button").disabled'),false);
  console.log('PASS: one-button UI, automatic timing, byte confirmation, authenticated requests, mobile layouts, changed-radio/CPU notices, errors, duplicate guard and hidden-page interruption');
 }finally{await page.close();}
})().catch(e=>{console.error(e);process.exitCode=1;});
