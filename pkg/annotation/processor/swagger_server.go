package processor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SwaggerServer Swagger文档服务器
type SwaggerServer struct {
	generator *SwaggerGenerator
	swaggerUI string
}

// NewSwaggerServer 创建新的Swagger服务器
func NewSwaggerServer(generator *SwaggerGenerator) *SwaggerServer {
	return &SwaggerServer{
		generator: generator,
		swaggerUI: getSwaggerUIPath(),
	}
}

// RegisterRoutes 注册Swagger相关路由
func (ss *SwaggerServer) RegisterRoutes(router *gin.Engine) {
	// Swagger UI页面
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, ss.getSwaggerHTML())
	})
	
	// Swagger JSON规范
	router.GET("/swagger/swagger.json", func(c *gin.Context) {
		ss.handleSwaggerJSON(c)
	})
	
	// 静态资源
	router.Static("/swagger/static", filepath.Join(ss.swaggerUI, "static"))
}

// handleSwaggerJSON 处理Swagger JSON请求
func (ss *SwaggerServer) handleSwaggerJSON(c *gin.Context) {
	// 生成Swagger规范
	spec, err := ss.generator.GenerateSwaggerSpec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate swagger spec: %v", err),
		})
		return
	}
	
	// 返回JSON
	c.JSON(http.StatusOK, spec)
}

// getSwaggerHTML 获取Swagger HTML页面
func (ss *SwaggerServer) getSwaggerHTML() string {
	// 读取HTML模板
	htmlPath := filepath.Join(ss.swaggerUI, "index.html")
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		// 如果文件不存在，返回默认的HTML内容
		return ss.getDefaultSwaggerHTML()
	}
	
	return string(content)
}

// getDefaultSwaggerHTML 获取默认的Swagger HTML内容
func (ss *SwaggerServer) getDefaultSwaggerHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>gRain Framework API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.3/swagger-ui.css" />
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
        }
        .swagger-ui .topbar {
            display: none;
        }
        #swagger-ui {
            margin-left: 300px;
            padding: 20px;
            box-sizing: border-box;
        }
        .nav-sidebar {
            width: 300px;
            background: #f8f9fa;
            border-right: 1px solid #dee2e6;
            padding: 20px;
            overflow-y: auto;
            position: fixed;
            left: 0;
            top: 0;
            bottom: 0;
            box-sizing: border-box;
            z-index: 100;
        }
        .nav-group {
            margin-bottom: 20px;
        }
        .nav-group-title {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            padding: 10px;
            background: #e9ecef;
            border-radius: 4px;
            margin-bottom: 10px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .nav-group-title:after {
            content: '▼';
            font-size: 12px;
            transition: transform 0.3s;
        }
        .nav-group.collapsed .nav-group-title:after {
            transform: rotate(-90deg);
        }
        .nav-group.collapsed .nav-items {
            display: none;
        }
        .nav-items {
            margin-left: 10px;
        }
        .nav-item {
            padding: 8px 12px;
            margin: 4px 0;
            cursor: pointer;
            border-radius: 4px;
            display: flex;
            align-items: center;
            transition: all 0.3s ease;
        }
        .nav-item:hover {
            background: #e9ecef;
        }
        .method-badge {
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 12px;
            font-weight: 600;
            margin-right: 8px;
            min-width: 50px;
            text-align: center;
            text-transform: uppercase;
        }
        .get { background: #61affe; color: white; }
        .post { background: #49cc90; color: white; }
        .put { background: #fca130; color: white; }
        .delete { background: #f93e3e; color: white; }
        .nav-item-text {
            flex: 1;
            font-size: 13px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .nav-item-description {
            font-size: 12px;
            color: #666;
            margin-top: 4px;
            display: none;
        }
        .nav-item:hover .nav-item-description {
            display: block;
            white-space: normal;
        }
        #api-search {
            width: 100%;
            padding: 8px;
            margin-bottom: 16px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        .swagger-ui .opblock-tag-section {
            margin-top: 20px;
        }
        .swagger-ui .opblock {
            margin: 0 0 15px;
            border: 1px solid rgba(59,65,81,.3);
            border-radius: 4px;
            box-shadow: 0 0 3px rgba(0,0,0,.1);
        }
        .swagger-ui .opblock.is-open .opblock-summary {
            border-bottom: 1px solid #e8e8e8;
        }
        .swagger-ui .opblock-tag {
            display: none !important;
        }
        .swagger-ui .opblock-tag-section {
            margin: 0 !important;
        }
        .swagger-ui .wrapper {
            padding: 0 !important;
        }
        .swagger-ui .opblock-tag-section h4 {
            display: none;
        }
    </style>
</head>
<body>
<div class="nav-sidebar">
    <h2>API 导航</h2>
    <input type="text" id="api-search" placeholder="搜索 API...">
    <div id="api-nav"></div>
</div>
<div id="swagger-ui"></div>

<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.3/swagger-ui-bundle.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.18.3/swagger-ui-standalone-preset.js"></script>
<script>
    let swaggerUI;

    // 添加重试函数
    function waitForElement(selector, maxAttempts = 10, interval = 500) {
        return new Promise((resolve, reject) => {
            let attempts = 0;

            const check = () => {
                attempts++;
                console.log(`Attempt ${attempts} to find element: ${selector}`);

                const element = document.querySelector(selector);
                if (element) {
                    console.log(`Element found: ${selector}`);
                    resolve(element);
                    return;
                }

                if (attempts >= maxAttempts) {
                    console.log(`Max attempts (${maxAttempts}) reached for: ${selector}`);
                    reject(new Error(`Element not found after ${maxAttempts} attempts: ${selector}`));
                    return;
                }

                setTimeout(check, interval);
            };

            check();
        });
    }

    // 添加查找操作块的函数
    async function findOperationBlock(api, tag) {
        console.log('Finding operation block for:', api);

        // 等待 Swagger UI 内容加载
        await waitForElement('.swagger-ui');

        // 尝试查找操作块
        const findBlock = () => {
            const operations = document.querySelectorAll('.opblock');
            console.log('Found operation blocks:', operations.length);

            // 记录所有找到的操作块信息
            operations.forEach(op => {
                console.log('Operation block:', {
                    method: op.getAttribute('data-method'),
                    path: op.getAttribute('data-path'),
                    className: op.className,
                    id: op.id,
                    text: op.textContent
                });
            });

            // 1. 尝试通过路径和方法匹配
            let targetOp = Array.from(operations).find(op => {
                const pathMatch = op.textContent.includes(api.path);
                const methodMatch = op.classList.contains(`opblock-${api.method.toLowerCase()}`);
                return pathMatch && methodMatch;
            });

            // 2. 如果没找到，尝试通过 ID 匹配
            if (!targetOp) {
                const possibleIds = [
                    `operations-${tag.replace(/\s+/g, '_')}-${api.operationId}`,
                    `operations-${api.method.toLowerCase()}-${api.path.replace(/[^w-]/g, '_')}`,
                    `operations-tag-${tag.replace(/\s+/g, '_')}`
                ];
                console.log('Trying possible IDs:', possibleIds);

                for (const id of possibleIds) {
                    targetOp = document.getElementById(id);
                    if (targetOp) {
                        console.log('Found by ID:', id);
                        break;
                    }
                }
            }

            return targetOp;
        };

        // 重试几次查找操作块
        for (let i = 0; i < 5; i++) {
            const block = findBlock();
            if (block) {
                // 先隐藏所有操作块
                document.querySelectorAll('.opblock').forEach(op => {
                    if (op !== block) {
                        op.style.display = 'none';
                    }
                });

                // 显示目标操作块
                block.style.display = 'block';

                // 确保父元素可见
                const parents = [];
                let parent = block.parentElement;
                while (parent && !parent.classList.contains('swagger-ui')) {
                    parents.push(parent);
                    parent = parent.parentElement;
                }
                parents.forEach(p => p.style.display = 'block');

                return block;
            }
            console.log(`Attempt ${i + 1} failed, waiting before retry...`);
            await new Promise(resolve => setTimeout(resolve, 1000));
        }

        throw new Error('Operation block not found after multiple attempts');
    }

    window.onload = function() {
        console.log('Window loaded, initializing Swagger UI...');

        // 添加重置显示的函数
        function resetDisplay() {
            document.querySelectorAll('.opblock').forEach(op => {
                op.style.display = 'block';
            });
            document.querySelectorAll('.opblock-tag-section').forEach(section => {
                section.style.display = 'block';
            });
        }

        // 添加搜索框的重置按钮
        const searchInput = document.getElementById('api-search');
        const resetButton = document.createElement('button');
        resetButton.textContent = '重置显示';
        resetButton.style.marginLeft = '10px';
        resetButton.onclick = resetDisplay;
        searchInput.parentNode.insertBefore(resetButton, searchInput.nextSibling);

        fetch('/swagger/swagger.json')
            .then(response => response.json())
            .then(spec => {
                console.log('Swagger spec loaded:', spec);
                swaggerUI = SwaggerUIBundle({
                    spec: spec,
                    dom_id: '#swagger-ui',
                    deepLinking: true,
                    presets: [
                        SwaggerUIBundle.presets.apis,
                        SwaggerUIStandalonePreset
                    ],
                    plugins: [
                        SwaggerUIBundle.plugins.DownloadUrl
                    ],
                    layout: "BaseLayout",
                    defaultModelsExpandDepth: -1,
                    displayRequestDuration: true,
                    docExpansion: "none",
                    filter: false,
                    defaultModelExpandDepth: -1,
                    defaultModelRendering: 'model',
                    showExtensions: false,
                    showCommonExtensions: false,
                    supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                    tryItOutEnabled: false,
                    displayOperationId: false,
                    onComplete: () => {
                        console.log('Swagger UI initialization completed');
                        buildNavigation(spec);

                        // 隐藏示例值和模型
                        const style = document.createElement('style');
                        style.textContent = `
                            .example,
                            .model-example,
                            .model-box,
                            .models,
                            .response-col_description__inner > .renderedMarkdown > p,
                            .response-col_description__inner > div > div {
                                display: none !important;
                            }
                            .opblock-description-wrapper,
                            .opblock-external-docs-wrapper,
                            .opblock-title_normal {
                                display: none !important;
                            }
                            .swagger-ui .responses-inner h4,
                            .swagger-ui .responses-inner h5 {
                                display: none !important;
                            }
                        `;
                        document.head.appendChild(style);
                    }
                });
            })
            .catch(error => console.error('Error loading swagger.json:', error));

        function buildNavigation(spec) {
            console.log('Building navigation...');
            const nav = document.getElementById('api-nav');
            const paths = spec.paths || {};
            const groups = {};

            // Group by tags
            Object.entries(paths).forEach(([path, methods]) => {
                Object.entries(methods).forEach(([method, operation]) => {
                    const tags = operation.tags || ['其他'];
                    tags.forEach(tag => {
                        if (!groups[tag]) {
                            groups[tag] = [];
                        }
                        groups[tag].push({
                            method: method.toUpperCase(),
                            path: path,
                            summary: operation.summary || path,
                            description: operation.description || '',
                            operationId: operation.operationId || `${method}_${path.replace(/[^w]/g, '_')}`
                        });
                    });
                });
            });

            console.log('Grouped APIs:', groups);

            // Create search functionality
            const searchInput = document.getElementById('api-search');
            searchInput.addEventListener('input', function(e) {
                const searchText = e.target.value.toLowerCase();
                document.querySelectorAll('.nav-item').forEach(item => {
                    const text = item.textContent.toLowerCase();
                    item.style.display = text.includes(searchText) ? 'flex' : 'none';

                    const group = item.closest('.nav-group');
                    if (text.includes(searchText)) {
                        group.classList.remove('collapsed');
                    }
                });
            });

            // Create navigation menu
            nav.innerHTML = '';
            Object.keys(groups).sort().forEach(tag => {
                const groupDiv = document.createElement('div');
                groupDiv.className = 'nav-group';

                const groupTitle = document.createElement('div');
                groupTitle.className = 'nav-group-title';
                groupTitle.textContent = tag;
                groupTitle.onclick = () => {
                    groupDiv.classList.toggle('collapsed');
                };

                const itemsDiv = document.createElement('div');
                itemsDiv.className = 'nav-items';

                groups[tag].forEach(api => {
                    const item = document.createElement('div');
                    item.className = 'nav-item';

                    const methodBadge = document.createElement('span');
                    methodBadge.className = `method-badge ${api.method.toLowerCase()}`;
                    methodBadge.textContent = api.method;

                    const textContainer = document.createElement('div');
                    textContainer.className = 'nav-item-text';

                    const title = document.createElement('div');
                    title.textContent = api.summary;

                    const description = document.createElement('div');
                    description.className = 'nav-item-description';
                    description.textContent = api.description;

                    textContainer.appendChild(title);
                    textContainer.appendChild(description);

                    item.appendChild(methodBadge);
                    item.appendChild(textContainer);

                    item.onclick = async () => {
                        console.log('API item clicked:', {
                            method: api.method,
                            path: api.path,
                            summary: api.summary
                        });

                        try {
                            const targetOp = await findOperationBlock(api, tag);
                            console.log('Found target operation:', targetOp);

                            if (targetOp) {
                                console.log('Scrolling to and expanding operation...');
                                targetOp.scrollIntoView({ behavior: 'smooth', block: 'center' });

                                // 确保展开标签组
                                const tagSection = targetOp.closest('.opblock-tag-section');
                                if (tagSection) {
                                    const tagButton = tagSection.querySelector('.opblock-tag');
                                    if (tagButton && !tagSection.classList.contains('is-open')) {
                                        console.log('Expanding tag section...');
                                        tagButton.click();
                                    }
                                }

                                // 展开操作块
                                setTimeout(() => {
                                    const summary = targetOp.querySelector('.opblock-summary');
                                    if (summary) {
                                        console.log('Found summary element:', summary);
                                        if (!targetOp.classList.contains('is-open')) {
                                            console.log('Clicking summary to expand...');
                                            try {
                                                // 尝试使用原生点击
                                                summary.click();
                                            } catch (e) {
                                                console.log('Native click failed, trying dispatch event...');
                                                // 如果原生点击失败，尝试派发点击事件
                                                summary.dispatchEvent(new MouseEvent('click', {
                                                    bubbles: true,
                                                    cancelable: true,
                                                    view: window
                                                }));
                                            }

                                            // 检查是否成功展开
                                            setTimeout(() => {
                                                if (!targetOp.classList.contains('is-open')) {
                                                    console.log('Operation block still not open, trying one more time...');
                                                    const button = summary.querySelector('button') || summary;
                                                    button.click();
                                                }
                                            }, 100);
                                        } else {
                                            console.log('Operation block already open');
                                        }
                                    } else {
                                        console.error('Summary element not found');
                                    }
                                }, 300); // 给足够的时间让滚动完成
                            }
                        } catch (error) {
                            console.error('Error finding operation block:', error);
                        }
                    };

                    itemsDiv.appendChild(item);
                });

                groupDiv.appendChild(groupTitle);
                groupDiv.appendChild(itemsDiv);
                nav.appendChild(groupDiv);
            });

            console.log('Navigation build completed');
        }
    }
</script>
</body>
</html>`
}

// getSwaggerUIPath 获取Swagger UI路径
func getSwaggerUIPath() string {
	// 尝试多个可能的路径
	possiblePaths := []string{
		"./swagger-ui",
		"./static/swagger-ui",
		"./public/swagger-ui",
		"./assets/swagger-ui",
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// 如果都找不到，返回当前目录
	return "."
}

// SaveSwaggerSpec 保存Swagger规范到文件
func (ss *SwaggerServer) SaveSwaggerSpec(outputPath string) error {
	spec, err := ss.generator.GenerateSwaggerSpec()
	if err != nil {
		return fmt.Errorf("failed to generate swagger spec: %w", err)
	}
	
	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// 序列化为JSON
	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal swagger spec: %w", err)
	}
	
	// 写入文件
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write swagger spec: %w", err)
	}
	
	return nil
} 