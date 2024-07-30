package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	//"os/notify"не стандартная библиотека

	//"os/exec"
	"sync"
	"syscall"

	"github.com/pircuser61/go_less/config"
)

const bufSize = 100

var l *slog.Logger

func fileListener(ctx context.Context, onExit func()) {
	defer onExit()

	fileInName := config.GetFileIn()
	fileIn, err := os.Open(fileInName)
	if err != nil {
		l.Error("file in error", slog.String("Msg", err.Error()))
		return
	}
	defer fileIn.Close()

	fileOutName := config.GetFileOut()
	fileOut, err := os.Create(fileOutName)
	if err != nil {
		l.Error("file out error", slog.String("Msg", err.Error()))
		return
	}

	defer func() {
		fileOut.Close()
		defer os.Remove(fileOutName)
	}()

	l.Info("Запуск fileListener...")

	buf := make([]byte, bufSize)

	stat, err := os.Stat(fileInName) //  не выводим то что уже было в файле до запуска программы
	if err != nil {
		l.Error("Ошибка получения данныйх о файле:",
			slog.String("file", fileInName),
			slog.String("Msg", err.Error()))
		return
	}
	prevSize := stat.Size()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		l.Debug("Проверка состояния файла...")
		stat, err := os.Stat(fileInName)
		if err != nil {
			l.Error("Ошибка получения данныйх о файле:",
				slog.String("file", fileInName),
				slog.String("Msg", err.Error()))
			return
		}
		if prevSize == stat.Size() {
			l.Debug("...файл без изменений")
			continue
		}
		l.Debug("...новые данные в файле")
		//вычитываем через буфер изменения
		for {
			n, err := fileIn.ReadAt(buf, prevSize)
			if err != nil && err != io.EOF {
				l.Error("Ошибка чтения из файла:",
					slog.String("Msg", err.Error()))
				return
			}

			fileOut.Write(buf[:n])
			prevSize += int64(n)
			if n != bufSize {
				break
			}
		}

		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
		}
	}
}

func sigListener(ctx context.Context, onExit func(), sigs chan os.Signal) {
	defer onExit()
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigs:
			if sig == syscall.SIGTERM {
				l.Info("Получен SIGTERM, остановка fileListener...")
				return
			} else {
				l.Info("", slog.String("Получен signal", sig.String()))
			}
		}
	}
}

func main() {

	l = slog.New(slog.NewTextHandler(os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	l.Info("Запуск приложения...")
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	wg := sync.WaitGroup{}

	wg.Add(2)
	onExit := func() {
		cancel()
		wg.Done()
	}

	go fileListener(ctx, onExit)
	go sigListener(ctx, onExit, sigs)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "ya.ru", "/t")
	} else {
		cmd = exec.CommandContext(ctx, "less", "+F", config.GetFileOut())
	}
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		l.Error("Программа "+cmd.Path+" завершилась с ошибкой:", slog.String("Msg", err.Error()))
	} else {
		l.Info("Программа " + cmd.Path + " отработала без ошибок")
	}
	cancel()
	l.Debug("Жду заверешения потоков... ")
	wg.Wait()
	l.Debug("...done")
}
