package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type renamedFile struct {
	from string
	to   string
}

func createDatabaseBackup(dbPath, backupPath string, allowPlaintextSecrets bool) (int64, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return 0, fmt.Errorf("database not found: %s", dbPath)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("database path is a directory: %s", dbPath)
	}

	db, err := openStore(dbPath, allowPlaintextSecrets)
	if err != nil {
		return 0, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.VacuumInto(backupPath); err != nil {
		return 0, err
	}

	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return 0, fmt.Errorf("stat backup: %w", err)
	}
	return backupInfo.Size(), nil
}

func restoreDatabaseFromBackup(dbPath, backupPath string, allowPlaintextSecrets bool) (int64, error) {
	if _, err := os.Stat(backupPath); err != nil {
		return 0, fmt.Errorf("backup file not found: %s", backupPath)
	}

	targetDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, fmt.Errorf("create database directory: %w", err)
	}

	restoreTmp := filepath.Join(targetDir, fmt.Sprintf(".niyantra-restore-%d.tmp", time.Now().UnixNano()))
	defer os.Remove(restoreTmp)
	defer removeSQLiteSidecars(restoreTmp)

	written, err := copyFile(backupPath, restoreTmp)
	if err != nil {
		return 0, fmt.Errorf("copy backup: %w", err)
	}

	// Validate on the temp copy so older backups can migrate safely without
	// mutating the original backup file.
	backupStore, err := openStore(restoreTmp, allowPlaintextSecrets)
	if err != nil {
		return 0, fmt.Errorf("invalid backup file: %w", err)
	}
	validatedPath := restoreTmp + ".validated"
	defer os.Remove(validatedPath)
	defer removeSQLiteSidecars(validatedPath)
	if err := backupStore.VacuumInto(validatedPath); err != nil {
		backupStore.Close()
		return 0, fmt.Errorf("rebuild validated backup: %w", err)
	}
	backupStore.Close()

	if err := replaceDatabaseFile(dbPath, validatedPath); err != nil {
		return 0, err
	}

	return written, nil
}

func copyFile(srcPath, dstPath string) (int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return 0, err
	}

	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return written, copyErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	return written, nil
}

func replaceDatabaseFile(dbPath, replacementPath string) error {
	stamp := time.Now().Format("20060102-150405")
	backupBase := dbPath + ".pre-restore-" + stamp
	moved := make([]renamedFile, 0, 3)

	moveIfExists := func(from, to string) error {
		if _, err := os.Stat(from); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if err := os.Rename(from, to); err != nil {
			return err
		}
		moved = append(moved, renamedFile{from: from, to: to})
		return nil
	}

	rollback := func() {
		for i := len(moved) - 1; i >= 0; i-- {
			_ = os.Rename(moved[i].to, moved[i].from)
		}
	}

	if err := moveIfExists(dbPath, backupBase); err != nil {
		return fmt.Errorf("move current database aside: %w", err)
	}
	if err := moveIfExists(dbPath+"-wal", backupBase+"-wal"); err != nil {
		rollback()
		return fmt.Errorf("move current database WAL aside: %w", err)
	}
	if err := moveIfExists(dbPath+"-shm", backupBase+"-shm"); err != nil {
		rollback()
		return fmt.Errorf("move current database shared memory aside: %w", err)
	}

	if err := os.Rename(replacementPath, dbPath); err != nil {
		rollback()
		return fmt.Errorf("replace database: %w", err)
	}

	for _, file := range moved {
		_ = os.Remove(file.to)
	}
	return nil
}

func removeSQLiteSidecars(dbPath string) {
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
}
