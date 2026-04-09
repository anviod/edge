// 代码高亮功能
function highlightCode() {
  const codeBlocks = document.querySelectorAll('pre code');
  codeBlocks.forEach(block => {
    // 简单的代码高亮实现
    let code = block.textContent;
    
    // 关键字高亮
    code = code.replace(/\b(const|let|var|function|if|else|for|while|return|class|import|export)\b/g, '<span style="color: #2980b9;">$1</span>');
    
    // 字符串高亮
    code = code.replace(/"([^"]*)"/g, '<span style="color: #27ae60;">"$1"</span>');
    code = code.replace(/'([^']*)'/g, '<span style="color: #27ae60;">\'$1\'</span>');
    
    // 数字高亮
    code = code.replace(/\b\d+\b/g, '<span style="color: #e74c3c;">$&</span>');
    
    block.innerHTML = code;
  });
}

// 添加复制按钮功能
function addCopyButtons() {
  const preBlocks = document.querySelectorAll('pre');
  preBlocks.forEach(pre => {
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.textContent = '复制';
    
    button.addEventListener('click', () => {
      const code = pre.querySelector('code').textContent;
      navigator.clipboard.writeText(code).then(() => {
        button.textContent = '已复制!';
        button.classList.add('copied');
        setTimeout(() => {
          button.textContent = '复制';
          button.classList.remove('copied');
        }, 2000);
      });
    });
    
    pre.appendChild(button);
  });
}

// 页面加载完成后执行
window.addEventListener('DOMContentLoaded', () => {
  highlightCode();
  addCopyButtons();
});