const items = document.querySelector('#items');
const message = document.querySelector('#message');
async function load() {
  const response = await fetch('/api/artifacts');
  const records = await response.json();
  items.innerHTML = records.map((item) => `<tr><td>${item.id}</td><td>${item.title}</td><td>${item.material}</td><td>${item.humidity}%</td><td>${item.status}</td><td><button data-id="${item.id}">送入修复</button></td></tr>`).join('');
  items.querySelectorAll('button').forEach((button) => button.addEventListener('click', async () => {
    const response = await fetch('/api/artifacts/status', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({id: button.dataset.id, status: 'treatment'})});
    message.textContent = response.ok ? '状态已更新' : '更新失败';
    load();
  }));
}
load();
