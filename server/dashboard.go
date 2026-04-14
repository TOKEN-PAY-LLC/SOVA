package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// Dashboard представляет веб-дашборд SOVA
type Dashboard struct {
	api       *ServerAPI
	startTime time.Time
}

// NewDashboard создает дашборд
func NewDashboard(api *ServerAPI) *Dashboard {
	return &Dashboard{
		api:       api,
		startTime: time.Now(),
	}
}

// RegisterDashboardRoutes регистрирует маршруты дашборда
func (d *Dashboard) RegisterDashboardRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", d.handleDashboard)
	mux.HandleFunc("/dashboard/api/stats", d.handleDashboardStats)
	mux.HandleFunc("/dashboard/api/connections", d.handleDashboardConnections)
	mux.HandleFunc("/dashboard/api/logs", d.handleDashboardLogs)
}

func (d *Dashboard) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := d.api.GetStats()
	conns := d.api.ConnMonitor.GetActiveConnections()

	data := map[string]interface{}{
		"uptime":             time.Since(d.startTime).String(),
		"uptime_seconds":     int(time.Since(d.startTime).Seconds()),
		"total_users":        stats.TotalUsers,
		"active_sessions":    stats.ActiveSessions,
		"active_connections": len(conns),
		"total_bytes":        stats.TotalBytes,
		"memory_alloc_mb":    float64(m.Alloc) / 1024 / 1024,
		"memory_sys_mb":      float64(m.Sys) / 1024 / 1024,
		"goroutines":         runtime.NumGoroutine(),
		"go_version":         runtime.Version(),
		"os_arch":            fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	json.NewEncoder(w).Encode(data)
}

func (d *Dashboard) handleDashboardConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	conns := d.api.ConnMonitor.GetActiveConnections()

	var connList []map[string]interface{}
	for id, conn := range conns {
		connList = append(connList, map[string]interface{}{
			"id":         id,
			"client_ip":  conn.ClientIP,
			"user_id":    conn.UserID,
			"duration":   time.Since(conn.StartTime).String(),
			"bytes_up":   conn.BytesUp,
			"bytes_down": conn.BytesDown,
		})
	}

	json.NewEncoder(w).Encode(connList)
}

func (d *Dashboard) handleDashboardLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	logs := d.api.Logger.GetLogs(50)

	var logList []map[string]interface{}
	for _, log := range logs {
		logList = append(logList, map[string]interface{}{
			"timestamp": log.Timestamp.Format("15:04:05"),
			"level":     log.Level,
			"message":   log.Message,
			"client_ip": log.ClientIP,
			"user_id":   log.UserID,
		})
	}

	json.NewEncoder(w).Encode(logList)
}

func (d *Dashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SOVA Network Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --bg:#0d0b1a;--surface:#161230;--surface2:#1e1845;--border:#2d2660;
  --purple:#7c3aed;--purple-light:#a78bfa;--purple-glow:#7c3aed80;
  --cyan:#06b6d4;--green:#22c55e;--red:#ef4444;--yellow:#eab308;
  --text:#e2e8f0;--text-dim:#94a3b8;
}
body{background:var(--bg);color:var(--text);font-family:'Segoe UI',system-ui,-apple-system,sans-serif;min-height:100vh;overflow-x:hidden}
.glow-bg{position:fixed;top:-200px;left:50%;transform:translateX(-50%);width:600px;height:600px;background:radial-gradient(circle,var(--purple-glow) 0%,transparent 70%);pointer-events:none;z-index:0;animation:pulse 4s ease-in-out infinite}
@keyframes pulse{0%,100%{opacity:.3;transform:translateX(-50%) scale(1)}50%{opacity:.6;transform:translateX(-50%) scale(1.1)}}
@keyframes fadeIn{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
@keyframes slideIn{from{opacity:0;transform:translateX(-20px)}to{opacity:1;transform:translateX(0)}}
@keyframes owlFloat{0%,100%{transform:translateY(0)}50%{transform:translateY(-8px)}}
@keyframes blink{0%,90%,100%{opacity:1}95%{opacity:0}}
.container{max-width:1400px;margin:0 auto;padding:20px;position:relative;z-index:1}
header{display:flex;align-items:center;gap:20px;padding:20px 0;border-bottom:1px solid var(--border);margin-bottom:30px;animation:fadeIn .6s ease-out}
.owl-icon{font-size:48px;animation:owlFloat 3s ease-in-out infinite}
.owl-eyes{display:inline-block;animation:blink 4s infinite}
h1{font-size:2rem;background:linear-gradient(135deg,var(--purple-light),var(--cyan));-webkit-background-clip:text;-webkit-text-fill-color:transparent;font-weight:800;letter-spacing:-0.5px}
.subtitle{color:var(--text-dim);font-size:.85rem;margin-top:2px}
.badge{display:inline-block;background:var(--purple);color:#fff;padding:2px 10px;border-radius:20px;font-size:.7rem;font-weight:600;margin-left:10px;animation:pulse 2s infinite}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:16px;margin-bottom:24px}
.card{background:var(--surface);border:1px solid var(--border);border-radius:16px;padding:20px;animation:fadeIn .6s ease-out;transition:all .3s ease;position:relative;overflow:hidden}
.card::before{content:'';position:absolute;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,var(--purple),var(--cyan));opacity:0;transition:opacity .3s}
.card:hover{border-color:var(--purple);transform:translateY(-2px);box-shadow:0 8px 32px var(--purple-glow)}.card:hover::before{opacity:1}
.card-title{color:var(--text-dim);font-size:.75rem;text-transform:uppercase;letter-spacing:1px;margin-bottom:8px}
.card-value{font-size:2rem;font-weight:700;color:var(--purple-light)}
.card-unit{font-size:.8rem;color:var(--text-dim);margin-left:4px}
.panel{background:var(--surface);border:1px solid var(--border);border-radius:16px;padding:20px;margin-bottom:24px;animation:fadeIn .8s ease-out}
.panel-title{font-size:1.1rem;font-weight:600;margin-bottom:16px;display:flex;align-items:center;gap:8px}
.panel-title .dot{width:8px;height:8px;border-radius:50%;background:var(--green);animation:pulse 2s infinite}
table{width:100%;border-collapse:collapse}
th{text-align:left;color:var(--text-dim);font-size:.75rem;text-transform:uppercase;letter-spacing:1px;padding:8px 12px;border-bottom:1px solid var(--border)}
td{padding:10px 12px;border-bottom:1px solid var(--border);font-size:.9rem;animation:slideIn .4s ease-out}
tr:hover td{background:var(--surface2)}
.log-level{padding:2px 8px;border-radius:4px;font-size:.7rem;font-weight:600}
.log-INFO{background:#22c55e20;color:var(--green)}
.log-WARN{background:#eab30820;color:var(--yellow)}
.log-ERROR{background:#ef444420;color:var(--red)}
.status-bar{position:fixed;bottom:0;left:0;right:0;background:var(--surface);border-top:1px solid var(--border);padding:8px 20px;display:flex;justify-content:space-between;font-size:.75rem;color:var(--text-dim);z-index:10}
.status-dot{width:6px;height:6px;border-radius:50%;background:var(--green);display:inline-block;margin-right:6px;animation:pulse 2s infinite}
.progress-ring{width:60px;height:60px;position:relative}
.progress-ring svg{transform:rotate(-90deg)}
.progress-ring circle{fill:none;stroke-width:4}
.progress-ring .bg{stroke:var(--border)}
.progress-ring .fg{stroke:var(--purple);stroke-linecap:round;transition:stroke-dashoffset .6s ease}
.progress-label{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);font-size:.8rem;font-weight:700;color:var(--purple-light)}
footer{text-align:center;padding:40px 0 60px;color:var(--text-dim);font-size:.8rem}
footer a{color:var(--purple-light);text-decoration:none}
@media(max-width:768px){.grid{grid-template-columns:1fr 1fr}h1{font-size:1.4rem}.card-value{font-size:1.4rem}}
@media(max-width:480px){.grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="glow-bg"></div>
<div class="container">
<header>
  <div class="owl-icon">
    <span class="owl-eyes">🦉</span>
  </div>
  <div>
    <h1>SOVA Network <span class="badge">v1.0.0</span></h1>
    <div class="subtitle">SOVA Proxy &mdash; SOVA Protocol &mdash; SOVA VPN control surface</div>
  </div>
</header>

<div class="grid" id="stats-grid">
  <div class="card" style="animation-delay:.1s">
    <div class="card-title">Uptime</div>
    <div class="card-value" id="stat-uptime">--</div>
  </div>
  <div class="card" style="animation-delay:.2s">
    <div class="card-title">Active Connections</div>
    <div class="card-value" id="stat-conns">0</div>
  </div>
  <div class="card" style="animation-delay:.3s">
    <div class="card-title">Total Users</div>
    <div class="card-value" id="stat-users">0</div>
  </div>
  <div class="card" style="animation-delay:.4s">
    <div class="card-title">Traffic</div>
    <div class="card-value" id="stat-traffic">0<span class="card-unit">B</span></div>
  </div>
  <div class="card" style="animation-delay:.5s">
    <div class="card-title">Memory</div>
    <div class="card-value" id="stat-memory">0<span class="card-unit">MB</span></div>
  </div>
  <div class="card" style="animation-delay:.6s">
    <div class="card-title">Goroutines</div>
    <div class="card-value" id="stat-goroutines">0</div>
  </div>
</div>

<div class="panel">
  <div class="panel-title"><span class="dot"></span> Active Connections</div>
  <table>
    <thead><tr><th>ID</th><th>Client IP</th><th>User</th><th>Duration</th><th>Upload</th><th>Download</th></tr></thead>
    <tbody id="conn-table"><tr><td colspan="6" style="text-align:center;color:var(--text-dim)">No active connections</td></tr></tbody>
  </table>
</div>

<div class="panel">
  <div class="panel-title"><span class="dot" style="background:var(--cyan)"></span> Recent Logs</div>
  <table>
    <thead><tr><th>Time</th><th>Level</th><th>Message</th><th>IP</th></tr></thead>
    <tbody id="log-table"><tr><td colspan="4" style="text-align:center;color:var(--text-dim)">No logs</td></tr></tbody>
  </table>
</div>

<div class="panel">
  <div class="panel-title"><span class="dot" style="background:var(--purple)"></span> System Info</div>
  <table>
    <tbody>
      <tr><td style="color:var(--text-dim)">Go Version</td><td id="sys-go">--</td></tr>
      <tr><td style="color:var(--text-dim)">OS / Arch</td><td id="sys-os">--</td></tr>
      <tr><td style="color:var(--text-dim)">System Memory</td><td id="sys-mem">--</td></tr>
    </tbody>
  </table>
</div>

<footer>
  <p>SOVA Protocol &copy; 2024-2026 &mdash; <a href="https://github.com/IvanChernykh/SOVA">GitHub</a></p>
  <p style="margin-top:4px">Autonomous AI Protocol for Internet Survival</p>
</footer>
</div>

<div class="status-bar">
  <div><span class="status-dot"></span>SOVA Server Online</div>
  <div id="status-time">--</div>
</div>

<script>
function formatBytes(b){
  if(b===0)return'0 B';
  const k=1024,s=['B','KB','MB','GB','TB'];
  const i=Math.floor(Math.log(b)/Math.log(k));
  return parseFloat((b/Math.pow(k,i)).toFixed(1))+' '+s[i];
}
function formatUptime(s){
  const h=Math.floor(s/3600),m=Math.floor((s%3600)/60),sec=s%60;
  if(h>0)return h+'h '+m+'m';
  if(m>0)return m+'m '+sec+'s';
  return sec+'s';
}

async function fetchStats(){
  try{
    const r=await fetch('/dashboard/api/stats');
    const d=await r.json();
    document.getElementById('stat-uptime').textContent=formatUptime(d.uptime_seconds||0);
    document.getElementById('stat-conns').textContent=d.active_connections||0;
    document.getElementById('stat-users').textContent=d.total_users||0;
    document.getElementById('stat-traffic').innerHTML=formatBytes(d.total_bytes||0);
    document.getElementById('stat-memory').innerHTML=(d.memory_alloc_mb||0).toFixed(1)+'<span class="card-unit">MB</span>';
    document.getElementById('stat-goroutines').textContent=d.goroutines||0;
    document.getElementById('sys-go').textContent=d.go_version||'--';
    document.getElementById('sys-os').textContent=d.os_arch||'--';
    document.getElementById('sys-mem').textContent=(d.memory_sys_mb||0).toFixed(1)+' MB';
  }catch(e){console.error('Stats fetch error:',e)}
}

async function fetchConnections(){
  try{
    const r=await fetch('/dashboard/api/connections');
    const d=await r.json();
    const t=document.getElementById('conn-table');
    if(!d||d.length===0){
      t.innerHTML='<tr><td colspan="6" style="text-align:center;color:var(--text-dim)">No active connections</td></tr>';
      return;
    }
    t.innerHTML=d.map(c=>'<tr><td>'+c.id+'</td><td>'+c.client_ip+'</td><td>'+c.user_id+'</td><td>'+c.duration+'</td><td>'+formatBytes(c.bytes_up)+'</td><td>'+formatBytes(c.bytes_down)+'</td></tr>').join('');
  }catch(e){console.error('Connections fetch error:',e)}
}

async function fetchLogs(){
  try{
    const r=await fetch('/dashboard/api/logs');
    const d=await r.json();
    const t=document.getElementById('log-table');
    if(!d||d.length===0){
      t.innerHTML='<tr><td colspan="4" style="text-align:center;color:var(--text-dim)">No logs</td></tr>';
      return;
    }
    t.innerHTML=d.slice(-20).reverse().map(l=>'<tr><td>'+l.timestamp+'</td><td><span class="log-level log-'+l.level+'">'+l.level+'</span></td><td>'+l.message+'</td><td>'+l.client_ip+'</td></tr>').join('');
  }catch(e){console.error('Logs fetch error:',e)}
}

function updateTime(){
  document.getElementById('status-time').textContent=new Date().toLocaleTimeString();
}

fetchStats();fetchConnections();fetchLogs();updateTime();
setInterval(fetchStats,3000);
setInterval(fetchConnections,5000);
setInterval(fetchLogs,5000);
setInterval(updateTime,1000);
</script>
</body>
</html>`
