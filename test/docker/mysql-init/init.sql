-- 初始化 PageSpy 数据库
CREATE DATABASE IF NOT EXISTS pagespy CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户（如果不存在）
CREATE USER IF NOT EXISTS 'pagespy'@'%' IDENTIFIED BY 'pagespy123';

-- 授权
GRANT ALL PRIVILEGES ON pagespy.* TO 'pagespy'@'%';
FLUSH PRIVILEGES;

USE pagespy;

-- 创建测试表（可选，GORM 会自动创建）
-- 这里只是为了验证数据库连接正常
SELECT 'PageSpy MySQL database initialized successfully!' as message;
