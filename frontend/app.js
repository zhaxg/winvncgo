import { GetStatus, RefreshID, Connect, CopyID } from './wailsjs/go/tasks/App.js';

// ── 状态更新 ──

function updateStatus(s) {
  document.getElementById('localIP').textContent = s.ip || '-';
  document.getElementById('currentId').textContent = s.id || '-';

  const vncOk = s.vnc === '运行中';
  document.getElementById('vncDot').className = 'dot ' + (vncOk ? 'dot-green' : 'dot-red');
  document.getElementById('vncStatus').textContent = s.vnc || '-';
  document.getElementById('vncStatus').style.color = vncOk ? '#16A34A' : '#DC2626';

  const redisOk = s.redis === '已连接';
  document.getElementById('redisDot').className = 'dot ' + (redisOk ? 'dot-green' : 'dot-red');
  document.getElementById('redisStatus').textContent = s.redis || '-';
  document.getElementById('redisStatus').style.color = redisOk ? '#16A34A' : '#DC2626';
}

// ── 初始化 ──

window.addEventListener('load', async function() {
  try {
    const status = await GetStatus();
    updateStatus(status);
  } catch (err) {
    console.error('获取状态失败:', err);
  }

  if (window.runtime) {
    window.runtime.EventsOn('status', function(s) {
      updateStatus(s);
    });
  }
});

// ── 操作函数 ──

async function refreshId() {
  try {
    const data = await RefreshID();
    if (data.ok) {
      showToast('ID 已刷新: ' + data.id);
    } else {
      showToast(data.error || '刷新失败');
    }
  } catch (err) {
    showToast('刷新失败: ' + err);
  }
}

async function copyId() {
  try {
    const id = await CopyID();
    if (id) {
      try {
        await navigator.clipboard.writeText(id);
        showToast('ID 已复制: ' + id);
      } catch {
        const ta = document.createElement('textarea');
        ta.value = id;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
        showToast('ID 已复制: ' + id);
      }
    }
  } catch (err) {
    showToast('复制失败: ' + err);
  }
}

async function connect() {
  const id = document.getElementById('remoteId').value.trim();
  const ip = document.getElementById('remoteIp').value.trim();

  if (!id) {
    showToast('请输入远程 ID');
    return;
  }

  const btn = document.querySelector('.btn-connect');
  btn.disabled = true;

  try {
    const data = await Connect(id, ip);
    if (data.ok) {
      showToast('已发起连接');
    } else {
      showToast(data.error || '连接失败');
    }
  } catch (err) {
    showToast('网络错误: ' + err);
  } finally {
    btn.disabled = false;
  }
}

// ── Toast ──

let toastTimer = null;
function showToast(msg) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove('show'), 2000);
}

// 暴露到全局（用于 onclick）
window.refreshId = refreshId;
window.copyId = copyId;
window.connect = connect;
