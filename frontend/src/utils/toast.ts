export function showToast(text: string, type: 'success' | 'error' | 'info' = 'info') {
  const toast = document.createElement('div')
  toast.textContent = text
  const colors = {
    success: '#07C160',
    error: '#FF4D4F',
    info: '#333',
  }
  toast.style.cssText = `
    position: fixed;
    top: 24px;
    left: 50%;
    transform: translateX(-50%);
    padding: 10px 24px;
    border-radius: 8px;
    font-size: 14px;
    z-index: 9999;
    color: #fff;
    background: ${colors[type]};
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    transition: opacity 0.3s;
    opacity: 1;
  `
  document.body.appendChild(toast)
  setTimeout(() => {
    toast.style.opacity = '0'
    setTimeout(() => toast.remove(), 300)
  }, 2000)
}
