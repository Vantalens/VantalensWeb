// 超级简单换页系统 - 绝对可靠版本
(function() {
    'use strict';
    
    // 全局变量
    var paginator = {
        currentPage: 1,
        totalPages: 1,
        mainElement: null,
        containerElement: null,
        controlsElement: null
    };
    
    // 等待页面完全加载
    function waitForLoad() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', initialize);
        } else {
            setTimeout(initialize, 100);
        }
    }
    
    // 初始化函数
    function initialize() {
        try {
            // 查找元素
            paginator.mainElement = document.querySelector('.main');
            paginator.containerElement = document.querySelector('.main > .container') || document.querySelector('.main .article-page');
            
            // 检查元素是否存在
            if (!paginator.mainElement) {
                return;
            }
            if (!paginator.containerElement) {
                return;
            }
            
            // 延迟执行确保所有内容都加载完成
            setTimeout(function() {
                setupPagination();
            }, 1000);
            
        } catch (error) {
            console.error('初始化错误:', error);
        }
    }
    
    // 设置换页功能
    function setupPagination() {
        try {
            if (!paginator.mainElement || !paginator.containerElement) {
                return;
            }
            
            // 获取尺寸信息
            var mainHeight = paginator.mainElement.clientHeight || 600;
            var containerHeight = paginator.containerElement.scrollHeight || 0;

            // 检查是否需要换页
            if (containerHeight <= mainHeight + 100) {
                return;
            }

            // 计算页数
            paginator.totalPages = Math.max(2, Math.ceil(containerHeight / mainHeight));

            // 创建控制按钮
            createControls();

        } catch (error) {
            console.error('设置换页错误:', error);
        }
    }
    
    // 创建控制按钮
    function createControls() {
        try {
            // 检查是否已存在
            if (document.getElementById('simple-page-nav')) {
                return;
            }
            
            // 创建控制容器
            var controls = document.createElement('div');
            controls.id = 'simple-page-nav';
            controls.style.cssText = [
                'position: fixed',
                'bottom: 50px',
                'left: 50%',
                'transform: translateX(-50%)',
                'z-index: 9999',
                'background: rgba(0, 0, 0, 0.8)',
                'padding: 12px 20px',
                'border-radius: 25px',
                'display: flex',
                'align-items: center',
                'gap: 15px',
                'font-family: Arial, sans-serif'
            ].join(';');
            
            // 上一个按钮
            var prevBtn = document.createElement('button');
            prevBtn.innerHTML = '◀';
            prevBtn.style.cssText = [
                'background: white',
                'border: none',
                'width: 45px',
                'height: 45px',
                'border-radius: 50%',
                'cursor: pointer',
                'font-size: 20px',
                'display: flex',
                'align-items: center',
                'justify-content: center',
                'transition: all 0.2s',
                'box-shadow: 0 2px 10px rgba(0,0,0,0.3)'
            ].join(';');
            prevBtn.onclick = function() { changePage(-1); };
            
            // 页码显示
            var pageInfo = document.createElement('span');
            pageInfo.id = 'page-info';
            pageInfo.style.cssText = [
                'color: white',
                'font-size: 16px',
                'font-weight: bold',
                'min-width: 80px',
                'text-align: center',
                'display: inline-block'
            ].join(';');
            pageInfo.textContent = paginator.currentPage + '/' + paginator.totalPages;
            
            // 下一个按钮
            var nextBtn = document.createElement('button');
            nextBtn.innerHTML = '▶';
            nextBtn.style.cssText = prevBtn.style.cssText;
            nextBtn.onclick = function() { changePage(1); };
            
            // 组装控件
            controls.appendChild(prevBtn);
            controls.appendChild(pageInfo);
            controls.appendChild(nextBtn);
            
            // 添加到页面
            document.body.appendChild(controls);
            paginator.controlsElement = controls;
            
            // 添加悬停效果
            setupButtonEffects(prevBtn, nextBtn);
            
            // 更新初始状态
            updateButtonStates(prevBtn, nextBtn);
            
            // 添加键盘支持
            document.addEventListener('keydown', function(e) {
                if (e.key === 'ArrowLeft') changePage(-1);
                if (e.key === 'ArrowRight') changePage(1);
            });

        } catch (error) {
            console.error('创建控件错误:', error);
        }
    }
    
    // 设置按钮效果
    function setupButtonEffects(prevBtn, nextBtn) {
        var buttons = [prevBtn, nextBtn];
        buttons.forEach(function(btn) {
            btn.onmouseover = function() {
                this.style.transform = 'scale(1.1)';
                this.style.boxShadow = '0 4px 15px rgba(0,0,0,0.4)';
            };
            btn.onmouseout = function() {
                this.style.transform = 'scale(1)';
                this.style.boxShadow = '0 2px 10px rgba(0,0,0,0.3)';
            };
        });
    }
    
    // 换页函数
    function changePage(direction) {
        try {
            var newPage = paginator.currentPage + direction;
            if (newPage < 1 || newPage > paginator.totalPages) {
                return;
            }
            
            paginator.currentPage = newPage;
            
            // 计算滚动位置
            var scrollAmount = (paginator.currentPage - 1) * (paginator.mainElement.clientHeight - 100);
            
            // 滚动
            if (paginator.mainElement.scrollTo) {
                paginator.mainElement.scrollTo({
                    top: scrollAmount,
                    behavior: 'smooth'
                });
            } else {
                // 兼容旧浏览器
                paginator.mainElement.scrollTop = scrollAmount;
            }
            
            // 更新显示
            var pageInfo = document.getElementById('page-info');
            if (pageInfo) {
                pageInfo.textContent = paginator.currentPage + '/' + paginator.totalPages;
            }
            
            // 更新按钮状态
            var buttons = paginator.controlsElement.getElementsByTagName('button');
            updateButtonStates(buttons[0], buttons[1]);

        } catch (error) {
            console.error('换页错误:', error);
        }
    }
    
    // 更新按钮状态
    function updateButtonStates(prevBtn, nextBtn) {
        try {
            // 上一个按钮状态
            if (prevBtn) {
                var isFirstPage = paginator.currentPage === 1;
                prevBtn.style.opacity = isFirstPage ? '0.3' : '1';
                prevBtn.style.cursor = isFirstPage ? 'not-allowed' : 'pointer';
            }
            
            // 下一个按钮状态
            if (nextBtn) {
                var isLastPage = paginator.currentPage === paginator.totalPages;
                nextBtn.style.opacity = isLastPage ? '0.3' : '1';
                nextBtn.style.cursor = isLastPage ? 'not-allowed' : 'pointer';
            }
        } catch (error) {
            console.error('更新按钮状态错误:', error);
        }
    }
    
    // 启动
    waitForLoad();
    
})();