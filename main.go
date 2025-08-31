package main

import (
"os"
"os/exec"
"path/filepath"
"fmt"
)

func main() {
// Получаем текущую директорию
dir, err := os.Getwd()
if err != nil {
fmt.Println("Ошибка получения текущей директории:", err)
os.Exit(1)
}

// Формируем путь к исполняемому файлу cmd/rsshub
cmdPath := filepath.Join(dir, "cmd", "rsshub", "rsshub")

// Проверяем существование исполняемого файла
if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
fmt.Println("Исполняемый файл не найден. Компилируем...")
buildCmd := exec.Command("go", "build", "-o", cmdPath, "./cmd/rsshub")
buildCmd.Stdout = os.Stdout
buildCmd.Stderr = os.Stderr

if err := buildCmd.Run(); err != nil {
fmt.Println("Ошибка компиляции:", err)
os.Exit(1)
}
}

// Получаем аргументы командной строки
args := os.Args[1:]

// Запускаем cmd/rsshub с переданными аргументами
cmd := exec.Command(cmdPath, args...)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Stdin = os.Stdin

// Выполняем команду и получаем код возврата
err = cmd.Run()
if err != nil {
if exitErr, ok := err.(*exec.ExitError); ok {
os.Exit(exitErr.ExitCode())
}
os.Exit(1)
}
}
