package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig содержит параметры подключения к базе данных
type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// MigrationConfig содержит конфигурацию миграции для одной базы данных
type MigrationConfig struct {
	Name   string   `json:"name"`
	Source DBConfig `json:"source"`
	Target DBConfig `json:"target"`
}

// Config содержит все конфигурации миграции
type Config struct {
	Migrations []MigrationConfig `json:"migrations"`
}

func main() {
	log.Println("===== Начало миграции данных PostgreSQL =====")
	log.Printf("Время начала: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// Загрузка конфигурации
	config, err := loadConfig("migration-tool/config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	log.Printf("Загружено %d миграций для выполнения\n", len(config.Migrations))

	// Выполнение миграций
	for i, migration := range config.Migrations {
		log.Printf("\n[%d/%d] Миграция: %s", i+1, len(config.Migrations), migration.Name)
		log.Println(strings.Repeat("=", 60))

		if err := migrateDatabases(migration); err != nil {
			log.Printf("❌ ОШИБКА при миграции %s: %v\n", migration.Name, err)
			log.Println("Продолжаем со следующей миграцией...")
			continue
		}

		log.Printf("✅ Миграция %s успешно завершена\n", migration.Name)
	}

	log.Println("\n===== Миграция данных завершена =====")
	log.Printf("Время окончания: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("не удалось разобрать JSON: %w", err)
	}

	return &config, nil
}

func buildConnectionString(cfg DBConfig, readOnly bool) string {
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.Username, cfg.Password,
	)

	if readOnly {
		connStr += " default_transaction_read_only=on"
	}

	return connStr
}

func migrateDatabases(migration MigrationConfig) error {
	ctx := context.Background()

	// Подключение к источнику (READ-ONLY)
	log.Printf("📥 Подключение к источнику: %s:%d/%s (READ-ONLY)...",
		migration.Source.Host, migration.Source.Port, migration.Source.Database)

	sourceConnStr := buildConnectionString(migration.Source, true)
	sourcePool, err := pgxpool.New(ctx, sourceConnStr)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к источнику: %w", err)
	}
	defer sourcePool.Close()

	// Проверка подключения к источнику
	if err := sourcePool.Ping(ctx); err != nil {
		return fmt.Errorf("не удалось проверить подключение к источнику: %w", err)
	}
	log.Println("✓ Подключение к источнику установлено")

	// Подключение к целевой БД
	log.Printf("📤 Подключение к целевой БД: %s:%d/%s...",
		migration.Target.Host, migration.Target.Port, migration.Target.Database)

	targetConnStr := buildConnectionString(migration.Target, false)
	targetPool, err := pgxpool.New(ctx, targetConnStr)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к целевой БД: %w", err)
	}
	defer targetPool.Close()

	// Проверка подключения к целевой БД
	if err := targetPool.Ping(ctx); err != nil {
		return fmt.Errorf("не удалось проверить подключение к целевой БД: %w", err)
	}
	log.Println("✓ Подключение к целевой БД установлено")

	// Получение списка таблиц из источника
	tables, err := getTables(ctx, sourcePool)
	if err != nil {
		return fmt.Errorf("не удалось получить список таблиц: %w", err)
	}

	if len(tables) == 0 {
		log.Println("⚠️  В базе данных нет таблиц для миграции")
		return nil
	}

	log.Printf("📋 Найдено таблиц: %d\n", len(tables))

	// Миграция каждой таблицы
	for i, table := range tables {
		log.Printf("\n[%d/%d] Миграция таблицы: %s", i+1, len(tables), table)

		if err := migrateTable(ctx, sourcePool, targetPool, table); err != nil {
			log.Printf("❌ Ошибка при миграции таблицы %s: %v", table, err)
			log.Println("⚠️  Продолжаем со следующей таблицы...")
			continue
		}

		log.Printf("✅ Таблица %s успешно мигрирована", table)
	}

	return nil
}

func getTables(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	query := `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY tablename
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

func migrateTable(ctx context.Context, source, target *pgxpool.Pool, tableName string) error {
	// Подсчет строк в источнике
	var sourceCount int64
	err := source.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&sourceCount)
	if err != nil {
		return fmt.Errorf("не удалось подсчитать строки в источнике: %w", err)
	}

	log.Printf("  📊 Строк в источнике: %d", sourceCount)

	if sourceCount == 0 {
		log.Println("  ℹ️  Таблица пуста, пропускаем")
		return nil
	}

	// Проверяем, существует ли таблица в целевой БД
	var targetTableExists bool
	err = target.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM pg_tables
			WHERE schemaname = 'public' AND tablename = $1
		)
	`, tableName).Scan(&targetTableExists)
	if err != nil {
		return fmt.Errorf("не удалось проверить существование таблицы в целевой БД: %w", err)
	}

	if !targetTableExists {
		log.Printf("  ⚠️  Таблица %s не существует в целевой БД, создаем структуру...", tableName)
		if err := copyTableStructure(ctx, source, target, tableName); err != nil {
			return fmt.Errorf("не удалось создать структуру таблицы: %w", err)
		}
		log.Println("  ✓ Структура таблицы создана")
	}

	// Получаем список колонок один раз
	var columnNames []string
	columnsQuery := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	colRows, err := source.Query(ctx, columnsQuery, tableName)
	if err != nil {
		return fmt.Errorf("не удалось получить список колонок: %w", err)
	}
	for colRows.Next() {
		var colName string
		if err := colRows.Scan(&colName); err != nil {
			colRows.Close()
			return fmt.Errorf("не удалось прочитать имя колонки: %w", err)
		}
		columnNames = append(columnNames, colName)
	}
	colRows.Close()

	if len(columnNames) == 0 {
		return fmt.Errorf("таблица %s не имеет колонок", tableName)
	}

	// Подготовка INSERT запроса
	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		pgx.Identifier{tableName}.Sanitize(),
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)

	// Миграция данных батчами для избежания таймаутов
	//const batchSize = 5 // Размер батча для чтения (для больших таблиц используем меньший размер)
	batchSize := int64(100)
	if tableName == "versions" {
		batchSize = 2
	}
	var totalInserted int64

	for offset := int64(0); offset < sourceCount; offset += batchSize {
		// Получение данных порциями с LIMIT/OFFSET
		query := fmt.Sprintf(
			"SELECT * FROM %s ORDER BY ctid LIMIT %d OFFSET %d",
			pgx.Identifier{tableName}.Sanitize(),
			batchSize,
			offset,
		)

		rows, err := source.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("не удалось выполнить SELECT (offset %d): %w", offset, err)
		}

		// Копирование данных из текущего батча
		batch := &pgx.Batch{}
		rowCount := 0

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				return fmt.Errorf("не удалось получить значения строки (offset %d): %w", offset, err)
			}

			batch.Queue(insertQuery, values...)
			rowCount++
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			return fmt.Errorf("ошибка при чтении строк (offset %d): %w", offset, err)
		}

		// Выполняем вставку батча
		if batch.Len() > 0 {
			results := target.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				_, err := results.Exec()
				if err != nil {
					results.Close()
					return fmt.Errorf("не удалось выполнить INSERT (offset %d): %w", offset, err)
				}
			}
			results.Close()
			totalInserted += int64(batch.Len())
		}

		log.Printf("  ⏳ Обработано: %d/%d строк (%.1f%%)",
			totalInserted, sourceCount, float64(totalInserted)/float64(sourceCount)*100)

		// Если получили меньше строк, чем ожидали - достигли конца таблицы
		if rowCount < int(batchSize) {
			break
		}
	}

	log.Printf("  ✓ Вставлено строк: %d", totalInserted)

	// Проверка количества строк в целевой БД
	var targetCount int64
	err = target.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&targetCount)
	if err != nil {
		return fmt.Errorf("не удалось подсчитать строки в целевой БД: %w", err)
	}

	log.Printf("  📊 Всего строк в целевой БД: %d", targetCount)

	return nil
}

func copyTableStructure(ctx context.Context, source, target *pgxpool.Pool, tableName string) error {
	// Получаем DDL создания таблицы
	query := `
		SELECT
			'CREATE TABLE ' || $1 || ' (' ||
			string_agg(
				column_name || ' ' || data_type ||
				CASE
					WHEN character_maximum_length IS NOT NULL
					THEN '(' || character_maximum_length || ')'
					ELSE ''
				END ||
				CASE WHEN is_nullable = 'NO' THEN ' NOT NULL' ELSE '' END,
				', '
			) || ')'
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		GROUP BY table_name
	`

	var createTableSQL string
	err := source.QueryRow(ctx, query, tableName).Scan(&createTableSQL)
	if err != nil {
		return fmt.Errorf("не удалось получить структуру таблицы: %w", err)
	}

	// Создаем таблицу в целевой БД
	_, err = target.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу: %w", err)
	}

	return nil
}
