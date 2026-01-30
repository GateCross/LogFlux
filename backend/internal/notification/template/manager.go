package template

import (
	"bytes"
	"context"
	"fmt"
	html_template "html/template"
	"sync"
	text_template "text/template"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"logflux/model"
)

// TemplateManager 管理通知模板的加载和渲染
type TemplateManager struct {
	db        *gorm.DB
	logger    logx.Logger
	templates map[string]*ParsedTemplate
	lock      sync.RWMutex
}

// ParsedTemplate 包含解析后的模板对象
type ParsedTemplate struct {
	Model *model.NotificationTemplate
	Text  *text_template.Template
	HTML  *html_template.Template
}

// NewTemplateManager 创建新的模板管理器
func NewTemplateManager(db *gorm.DB) *TemplateManager {
	return &TemplateManager{
		db:        db,
		logger:    logx.WithContext(context.Background()),
		templates: make(map[string]*ParsedTemplate),
	}
}

// LoadTemplates 从数据库加载所有模板到内存
func (tm *TemplateManager) LoadTemplates() error {
	var dbTemplates []model.NotificationTemplate
	// 尝试从数据库加载，如果表不存在（migration未完成），则忽略错误
	if err := tm.db.Find(&dbTemplates).Error; err != nil {
		tm.logger.Errorf("failed to load templates from db: %v", err)
		// Don't return error, just load defaults
	}

	tm.lock.Lock()
	defer tm.lock.Unlock()

	// 重置缓存
	tm.templates = make(map[string]*ParsedTemplate)

	// 加载数据库模板
	for _, t := range dbTemplates {
		if err := tm.parseAndCache(t); err != nil {
			tm.logger.Errorf("failed to parse template %s: %v", t.Name, err)
			continue
		}
	}

	// 加载默认模板（如果数据库中没有）
	tm.ensureDefaults()

	tm.logger.Infof("Loaded %d notification templates", len(tm.templates))
	return nil
}

// parseAndCache 解析模板字符串并缓存
func (tm *TemplateManager) parseAndCache(t model.NotificationTemplate) error {
	parsed := &ParsedTemplate{
		Model: &t,
	}

	var err error
	if t.Format == "html" {
		parsed.HTML, err = html_template.New(t.Name).Parse(t.Content)
	} else {
		// text, markdown, json 都使用 text/template 处理
		parsed.Text, err = text_template.New(t.Name).Parse(t.Content)
	}

	if err != nil {
		return err
	}

	tm.templates[t.Name] = parsed
	return nil
}

// Render 渲染指定名称的模板
func (tm *TemplateManager) Render(name string, data interface{}) (string, error) {
	tm.lock.RLock()
	tmpl, ok := tm.templates[name]
	tm.lock.RUnlock()

	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}

	var buf bytes.Buffer
	var err error

	if tmpl.HTML != nil {
		err = tmpl.HTML.Execute(&buf, data)
	} else if tmpl.Text != nil {
		err = tmpl.Text.Execute(&buf, data)
	} else {
		return "", fmt.Errorf("template %s has no valid parsed content", name)
	}

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderContent 直接渲染内容
func (tm *TemplateManager) RenderContent(content, format string, data interface{}) (string, error) {
	var buf bytes.Buffer
	var err error

	if format == "html" {
		tmpl, parseErr := html_template.New("preview").Parse(content)
		if parseErr != nil {
			return "", parseErr
		}
		err = tmpl.Execute(&buf, data)
	} else {
		// text, markdown, json
		tmpl, parseErr := text_template.New("preview").Parse(content)
		if parseErr != nil {
			return "", parseErr
		}
		err = tmpl.Execute(&buf, data)
	}

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ensureDefaults 确保默认模板存在
func (tm *TemplateManager) ensureDefaults() {
	defaults := GetDefaultTemplates()
	for _, def := range defaults {
		if _, exists := tm.templates[def.Name]; !exists {
			// 创建内存模型
			modelTmpl := model.NotificationTemplate{
				Name:    def.Name,
				Format:  def.Format,
				Content: def.Content,
				Type:    "system",
			}

			// 尝试解析并缓存
			if err := tm.parseAndCache(modelTmpl); err != nil {
				tm.logger.Errorf("failed to load default template %s: %v", def.Name, err)
				continue
			}

			// 尝试写入数据库（如果表存在）
			// 我们在一个单独的协程中做这个，以免阻塞启动，且忽略错误（例如重复键或表不存在）
			go func(t model.NotificationTemplate) {
				if err := tm.db.FirstOrCreate(&t, model.NotificationTemplate{Name: t.Name}).Error; err != nil {
					// 仅记录 debug 日志，因为如果是因为已存在（虽然我们检查了缓存但并发情况下可能）或其他原因，我们不希望太吵
					// tm.logger.Debugf("failed to persist default template %s: %v", t.Name, err)
				}
			}(modelTmpl)
		}
	}
}

// DefaultTemplateDef 默认模板定义结构
type DefaultTemplateDef struct {
	Name    string
	Format  string // html, text, markdown
	Content string
}

// GetDefaultTemplates 返回内置的默认模板列表
func GetDefaultTemplates() []DefaultTemplateDef {
	return []DefaultTemplateDef{
		{
			Name:   "default_email",
			Format: "html",
			Content: `<!DOCTYPE html>
<html>
<head>
<style>
  body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
  .container { max-width: 600px; margin: 0 auto; padding: 20px; }
  .header { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
  .content { margin-bottom: 20px; }
  .footer { font-size: 12px; color: #6c757d; margin-top: 30px; border-top: 1px solid #dee2e6; padding-top: 10px; }
  pre { background-color: #f8f9fa; padding: 10px; border-radius: 5px; overflow-x: auto; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <h2>[{{.Level}}] {{.Title}}</h2>
    <p>Time: {{.Timestamp}}</p>
  </div>
  <div class="content">
    <p>{{.Message}}</p>
    {{if .Data}}
    <h3>Details:</h3>
    <pre>{{.Data}}</pre>
    {{end}}
  </div>
  <div class="footer">
    <p>Sent by LogFlux Notification System</p>
  </div>
</div>
</body>
</html>`,
		},
		{
			Name:   "default_markdown",
			Format: "markdown",
			Content: `**[{{.Level}}] {{.Title}}**

**Time:** {{.Timestamp}}

**Message:**
{{.Message}}

{{if .Data}}
**Details:**
` + "```json" + `
{{.Data}}
` + "```" + `
{{end}}
`,
		},
		{
			Name:   "default_text",
			Format: "text",
			Content: `[{{.Level}}] {{.Title}}
Time: {{.Timestamp}}

Message: {{.Message}}

{{if .Data}}Details: {{.Data}}{{end}}
`,
		},
	}
}

// RenderContent 渲染模板内容字符串 (静态辅助函数)
func RenderContent(format string, content string, data interface{}) (string, error) {
	var buf bytes.Buffer
	var err error

	if format == "html" {
		tmpl, parseErr := html_template.New("preview").Parse(content)
		if parseErr != nil {
			return "", parseErr
		}
		err = tmpl.Execute(&buf, data)
	} else {
		// text, markdown, json
		tmpl, parseErr := text_template.New("preview").Parse(content)
		if parseErr != nil {
			return "", parseErr
		}
		err = tmpl.Execute(&buf, data)
	}

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
