'use strict';
'require baseclass';

const median=values=>{const v=values.slice().sort((a,b)=>a-b),i=Math.floor(v.length/2);return v.length%2?v[i]:(v[i-1]+v[i])/2;};
function stable(values){if(values.length<3)return false;const recent=values.slice(-3),mean=recent.reduce((a,b)=>a+b,0)/3;return mean>0&&(Math.max(...recent)-Math.min(...recent))/mean<=.1;}
function finished(elapsed,samples,timing){return elapsed>=timing.maxMs||(elapsed>=timing.minMs&&stable(samples));}
function mount(host,options){
 const timing=Object.assign({warmupMs:1000,minMs:6000,maxMs:10000,timeoutMs:18000},options.timing);
 const section=document.createElement('section');section.className='nst';
 section.innerHTML='<h2>Speed Test</h2><div class="cbi-section"><p class="nst-connection">This device &harr; router</p><table class="table"><tr class="tr table-titles"><th class="th">Measurement</th><th class="th">Result</th></tr><tr class="tr"><td class="td" data-title="Measurement">Download</td><td class="td" data-title="Result"><span class="nst-value" data-result="download">&mdash;</span> Mbps</td></tr><tr class="tr"><td class="td" data-title="Measurement">Upload</td><td class="td" data-title="Result"><span class="nst-value" data-result="upload">&mdash;</span> Mbps</td></tr><tr class="tr"><td class="td" data-title="Measurement">Latency</td><td class="td" data-title="Result"><span class="nst-value" data-result="latency">&mdash;</span> ms</td></tr></table><div class="cbi-progressbar" hidden role="progressbar" aria-label="Test progress" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"><div></div></div><p class="nst-status" role="status" aria-live="polite">Ready.</p><p class="nst-actions"><button type="button" class="cbi-button cbi-button-action">Start</button></p><div class="cbi-section-descr nst-help"><p>Keep this page visible until the test finishes. Timing adjusts automatically.</p><p>Results include this device, its connection and the router. Latency includes the router&rsquo;s response time.</p></div><p class="nst-warning alert-message warning" hidden></p></div>';
 host.append(section);
 const button=section.querySelector('button'),status=section.querySelector('.nst-status'),progress=section.querySelector('.cbi-progressbar'),connection=section.querySelector('.nst-connection'),warning=section.querySelector('.nst-warning');
 const setProgress=value=>{const percent=Math.round(value);progress.firstElementChild.style.width=percent+'%';progress.setAttribute('aria-valuenow',percent);progress.title=percent+'%';};
 const output=name=>section.querySelector('[data-result="'+name+'"]');
 const endpoint=options.endpoint||'/cgi-bin/network-speed-test';
 let active=false,cancelled='',controller=null,started=false,disposed=false;
 function check(){if(cancelled)throw Error(cancelled);}
 async function request(op,body){
  check();controller=new AbortController();const current=controller;
  const timer=setTimeout(()=>current.abort(),timing.timeoutMs);
  try {
   const response=await fetch(endpoint+'?op='+op,{method:'POST',credentials:'same-origin',cache:'no-store',headers:Object.assign({Authorization:'Bearer '+options.session},body?{'Content-Type':'application/octet-stream'}:{}),body,signal:current.signal});
   if(!response.ok){const data=await response.json().catch(()=>({}));throw Error(data.error||'The router could not complete this request.');}
   const data=await response.json();check();return data;
  } catch(e){check();if(e.name==='AbortError')throw Error('The connection timed out. Try again when the connection is stable.');throw e;}
  finally{clearTimeout(timer);if(controller===current)controller=null;}
 }
 async function finish(){
  if(!started)return null;
  // A cancelled stream can take a moment to release the router's transfer lock.
  for(let attempt=0;attempt<12;attempt++){
   const c=new AbortController(),timer=setTimeout(()=>c.abort(),3000);
   try{
    const response=await fetch(endpoint+'?op=finish',{method:'POST',credentials:'same-origin',cache:'no-store',keepalive:true,headers:{Authorization:'Bearer '+options.session},signal:c.signal});
    if(response.ok){started=false;return await response.json();}
    if(response.status!==409)break;
   }catch(e){break;}finally{clearTimeout(timer);}
   await new Promise(resolve=>setTimeout(resolve,250));
  }
  started=false;return null;
 }
 function showConnection(c){connection.textContent=c&&c.band?c.band+(c.channel?' · Channel '+c.channel:'')+' · This device ↔ router':'This device ↔ router';}
 async function latency(){
  status.textContent='Measuring latency…';const samples=[];
  for(let i=0;i<9;i++){const start=performance.now();await request('ping');if(i>=2)samples.push(performance.now()-start);setProgress(5+i);}
  output('latency').textContent=median(samples).toFixed(1);
 }
 async function download(){
  status.textContent='Measuring download…';controller=new AbortController();const current=controller;
  const timer=setTimeout(()=>current.abort(),timing.timeoutMs);let reader;
  try{
   const response=await fetch(endpoint+'?op=download',{method:'POST',credentials:'same-origin',cache:'no-store',headers:{Authorization:'Bearer '+options.session},signal:current.signal});
   if(!response.ok){const error=await response.json().catch(()=>({}));throw Error(error.error||'Download failed.');}
   if(!response.body)throw Error('This browser cannot measure a streaming download.');
   reader=response.body.getReader();let first=null,begin=null,windowStart=0,bytes=0,windowBytes=0,elapsed=0;const samples=[];
   for(;;){
    check();const part=await reader.read(),now=performance.now();if(part.done)break;
    if(first===null)first=now;
    if(now-first<timing.warmupMs)continue;
    if(begin===null){begin=windowStart=now;continue;}
    bytes+=part.value.byteLength;windowBytes+=part.value.byteLength;elapsed=now-begin;
    if(now-windowStart>=Math.min(1000,timing.minMs/3)){
     samples.push(windowBytes*8/(now-windowStart)/1000);windowBytes=0;windowStart=now;
     output('download').textContent=(bytes*8/elapsed/1000).toFixed(1);
     setProgress(15+Math.min(35,elapsed/timing.maxMs*35));
    }
    if(finished(elapsed,samples,timing))break;
   }
   if(bytes===0||elapsed<timing.minMs)throw Error('Download ended before a reliable measurement was possible. Try again.');
   output('download').textContent=(bytes*8/elapsed/1000).toFixed(1);
  }catch(e){check();if(e.name==='AbortError')throw Error('Download timed out. Try again when the connection is stable.');throw e;}
  finally{clearTimeout(timer);current.abort();if(reader)await reader.cancel().catch(()=>{});if(controller===current)controller=null;}
  setProgress(50);
 }
 async function upload(){
  status.textContent='Measuring upload…';
  const data=new Uint8Array(32*1024*1024),seed=new Uint8Array(65536);
  if(globalThis.crypto&&crypto.getRandomValues)crypto.getRandomValues(seed);else for(let i=0;i<seed.length;i++)seed[i]=Math.random()*256;
  for(let i=0;i<data.length;i+=seed.length)data.set(seed,i);
  let size=65536,begin=null,bytes=0,elapsed=0,rounds=0,busyRetries=0;const warmupStart=performance.now(),samples=[];
  for(;;){
   check();const before=performance.now();let response;
   try{response=await request('upload',data.subarray(0,size));}
   catch(e){if(rounds===0&&busyRetries++<12&&/Another transfer/.test(e.message)){await new Promise(resolve=>setTimeout(resolve,250));continue;}throw e;}
   const now=performance.now(),duration=now-before;rounds++;
   if(response.bytes!==size)throw Error('The router did not receive the complete upload.');
   if(begin!==null){bytes+=size;elapsed=now-begin;samples.push(size*8/duration/1000);output('upload').textContent=(bytes*8/elapsed/1000).toFixed(1);setProgress(50+Math.min(45,elapsed/timing.maxMs*45));}
   if(begin===null&&now-warmupStart>=timing.warmupMs&&rounds>=2)begin=now;
   if(begin!==null&&finished(elapsed,samples,timing))break;
   size=Math.max(65536,Math.min(data.length,size*4,Math.round(size/Math.max(1,duration)*750/65536)*65536));
  }
 }
 async function run(){
  if(active||disposed)return;
  active=true;cancelled='';button.disabled=true;progress.hidden=false;setProgress(0);warning.hidden=true;warning.textContent='';status.classList.remove('alert-message','error');section.setAttribute('aria-busy','true');
  for(const name of ['download','upload','latency'])output(name).textContent='—';
  let succeeded=false;
  try{
   status.textContent='Preparing test…';const result=await request('start');started=true;showConnection(result.connection);
   await latency();await download();await upload();check();
   status.textContent='Finishing…';const end=await finish();check();
   if(!end)throw Error('Could not verify the end of the test. Press Start to repeat it.');
   const notices=[];
   if(end.connection_changed)notices.push('The radio or channel changed during the test. Repeat it for a consistent comparison.');
   if(end.router_cpu_busy>=90)notices.push('The router was busy during the test and may have limited the result.');
   warning.textContent=notices.join(' ');warning.hidden=!notices.length;
   setProgress(100);status.textContent='Complete.';succeeded=true;
  }catch(e){status.textContent=e.message;status.classList.add('alert-message','error');}
  finally{
   if(started)await finish();active=false;button.disabled=false;section.removeAttribute('aria-busy');
   if(!succeeded){progress.hidden=true;for(const name of ['download','upload','latency'])output(name).textContent='—';}
  }
 }
 function cancel(){if(active){cancelled='Test interrupted. Keep this page visible and press Start to try again.';if(controller)controller.abort();}}
 const visibility=()=>{if(document.hidden)cancel();};document.addEventListener('visibilitychange',visibility);window.addEventListener('pagehide',cancel);
 const observer=new MutationObserver(()=>{if(!host.isConnected){disposed=true;cancel();observer.disconnect();document.removeEventListener('visibilitychange',visibility);window.removeEventListener('pagehide',cancel);}});
 // LuCI attaches the returned view after render() resolves.
 queueMicrotask(()=>{if(host.isConnected)observer.observe(document.body,{childList:true,subtree:true});else setTimeout(()=>{if(host.isConnected)observer.observe(document.body,{childList:true,subtree:true});},0);});
 button.addEventListener('click',run);
 return {run,cancel};
}
return baseclass.extend({mount:mount});
