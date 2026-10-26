package caddy

import (
	"context"

	"logflux/internal/utils/safego"
	systemmodel "logflux/model/system"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 历史日志字段回填的单批行数。
const logFieldBackfillBatchSize = 1000

// 回填完成标记在 system_configs 里的存储键。
const logFieldBackfillDoneKey = "caddy_log_field_backfill_done"

// BackfillLogFields 把历史日志遗漏的 level / logger 从 raw_log 一次性补回。
//
// 单条 SQL 分批推进：raw_log 为 JSON 字符串标量的是旧 console 文本行（全是访问日志）；
// 含 transaction 键的是 Coraza 审计行；其余对象行从 raw_log 取 level / logger，取不到按访问日志兜底。
// 全部完成后置位完成标记；中途失败不置位，下次启动重跑（幂等）。
// jsonb 的 ? 操作符与 GORM 占位符冲突，JSON 判断一律用 jsonb_exists 函数形式。
func BackfillLogFields(db *gorm.DB) {
	if db == nil {
		return
	}

	cfgModel := systemmodel.NewSystemConfigModel(db)
	if _, err := cfgModel.GetByKey(logFieldBackfillDoneKey); err == nil {
		return
	}

	safego.New(context.Background(), "日志级别字段回填").Go(func() {
		total := 0
		for {
			result := db.Exec(`
WITH batch AS (SELECT id FROM caddy_logs WHERE logger IS NULL OR logger = '' ORDER BY id LIMIT ?)
UPDATE caddy_logs SET
  level = CASE WHEN jsonb_typeof(raw_log) = 'string' THEN 'info'
               WHEN jsonb_exists(raw_log, 'transaction') THEN 'info'
               ELSE COALESCE(NULLIF(raw_log->>'level', ''), 'info') END,
  logger = CASE WHEN jsonb_typeof(raw_log) = 'string' THEN 'http.log.access'
                WHEN jsonb_exists(raw_log, 'transaction') THEN 'http.log.access'
                ELSE COALESCE(NULLIF(raw_log->>'logger', ''), 'http.log.access') END
FROM batch WHERE caddy_logs.id = batch.id`, logFieldBackfillBatchSize)
			if result.Error != nil {
				logx.Errorf("日志级别字段回填执行失败，未置位完成标记，下次启动将重跑: %v", result.Error)
				return
			}
			if result.RowsAffected == 0 {
				break
			}
			total += int(result.RowsAffected)
		}

		if err := cfgModel.SetByKey(logFieldBackfillDoneKey, "done"); err != nil {
			logx.Errorf("写入日志级别字段回填完成标记失败: %v", err)
			return
		}
		if total > 0 {
			logx.Infof("日志级别字段回填完成，共补齐 %d 条", total)
		}
	})
}
