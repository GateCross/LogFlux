package template

import (
	"bytes"
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
	tm := &TemplateManager{
		db:        db,
		templates: make(map[string]*ParsedTemplate),
	}
	return tm
}

// LoadTemplates 从数据库加载所有模板到内存
func (tm *TemplateManager) LoadTemplates() error {
	var dbTemplates []model.NotificationTemplate
	if err := tm.db.Find(&dbTemplates).Error; err != nil {
		return err
	}

	tm.lock.Lock()
	defer tm.lock.Unlock()

	// 重置缓存
	tm.templates = make(map[string]*ParsedTemplate)

	// 加载数据库模板
	for _, t := range dbTemplates {
		if err := tm.parseAndCache(t); err != nil {
			logx.Errorf("failed to parse template %s: %v", t.Name, err)
			continue
		}
	}

	// 加载默认模板（如果数据库中没有）
	tm.ensureDefaults()

	logx.Infof("loaded %d notification templates", len(tm.templates))
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

// ensureDefaults 确保默认模板存在
func (tm *TemplateManager) ensureDefaults() {
	defaults := GetDefaultTemplates()
	for _, def := range defaults {
		if _, exists := tm.templates[def.Name]; !exists {
			// 如果内存中不存在（即数据库也没加载到），则使用默认值
			// 注意：这里我们只在内存中加载，不强制写入数据库，除非需要
			// 为了简单起见，这里作为 fallback 存在内存中
			modelTmpl := model.NotificationTemplate{
				Name:    def.Name,
				Format:  def.Format,
				Content: def.Content,
				Type:    "system",
			}
			if err := tm.parseAndCache(modelTmpl); err != nil {
				logx.Errorf("failed to load default template %s: %v", def.Name, err)
			}
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
<body>
    <h2>LogFlux Notification: {{.Type}}</h2>
    <p><strong>Level:</strong> {{.Level}}</p>
    <p><strong>Time:</strong> {{.Time}}</p>
    <p><strong>Message:</strong></p>
    <pre>{{.Message}}</pre>
    {{if .Data}}
    <h3>Details:</h3>
    <ul>
        {{range $key, $value := .Data}}
        <li><strong>{{$key}}:</strong> {{$value}}</li>
        {{end}}
    </ul>
    {{end}}
</body>
</html>`,
		},
		{
			Name:   "default_telegram",
			Format: "markdown", // MarkdownV2
			Content: `*LogFlux Alert*
*Type*: {{.Type}}
*Level*: {{.Level}}
*Time*: {{.Time}}

*Message*:
{{.Message}}
`,
		},
		{
			Name:    "default_webhook",
			Format:  "json", // Note: Provider typically handles JSON structure, this is for payload body content if customizable
			Content: `{{.Message}}`,
		},
	}
}

// RenderContent 渲染模板内容字符串
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
		// text or markdown
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
