-- 002: 业主撤销报修工单闭环
-- GORM AutoMigrate 会自动补列；本脚本用于既有 MySQL 环境的版本化迁移。
-- cancelled 为业主专属终态：工单保留、不计入待处理数量，与 done/closed 并列。
ALTER TABLE `repairs`
    ADD COLUMN `cancelled_at` DATETIME(3) NULL AFTER `rating`;

CREATE INDEX `idx_repairs_cancelled_at` ON `repairs` (`cancelled_at`);
