package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pircuser61/go_less/config"
)

const bufSize = 100
const fileCheckTimeout = time.Second

var l *slog.Logger

// раз в fileCheckTimeout проверяет файл indata.txt, при изменении размера
// дописывает новые данные в outdata.txt
// можно было бы взять "os/notify" но это не стандартная библиотека
// как еще подписасться на обновления файлов - не знаю
func fileListener(ctx context.Context) {
	fileInName := config.GetFileIn()
	fileIn, err := os.Open(fileInName)
	if err != nil {
		l.Error("Ошибка открытия файла In", slog.String("Msg", err.Error()))
		return
	}
	defer fileIn.Close()

	fileOutName := config.GetFileOut()
	fileOut, err := os.Create(fileOutName)
	if err != nil {
		l.Error("Ошибка при создании файла Out", slog.String("Msg", err.Error()))
		return
	}

	defer func() {
		fileOut.Close()
		defer os.Remove(fileOutName)
	}()

	l.Debug("Запуск fileListener...")

	stat, err := os.Stat(fileInName) //  не выводим то что уже было в файле до запуска программы
	if err != nil {
		l.Error("Ошибка получения данныйх о файле:",
			slog.String("file", fileInName),
			slog.String("Msg", err.Error()))
		return
	}
	prevSize := stat.Size()

	buf := make([]byte, bufSize)

	ticker := time.NewTicker(fileCheckTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

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
	}
}

// пишет сигналы в лог, по SIGTERM завершает работу
func sigListener(ctx context.Context, sigs chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigs:
			l.Info("", slog.String("Получен signal", sig.String()))
			if sig == syscall.SIGTERM /*|| sig == os.Interrupt нужен что бы выйти из less */ {
				l.Info("остановка по сигналу...")
				return
			}
		}
	}
}

func main() {
	logOpts := slog.HandlerOptions{Level: slog.LevelInfo}
	l = slog.New(slog.NewTextHandler(os.Stdout, &logOpts))

	l.Info("Запуск приложения...")
	ctx, cancel := context.WithCancel(context.Background())

	// для остановки приложения можно было использовать что то вроде
	// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// но я не понял, что значит :
	//           "осуществляет запись сообщений в системный журнал ...в том числе по сигналам."
	// обрабатываю указанные сигналы
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt)

	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		fileListener(ctx)
		cancel()
		wg.Done()
	}()
	go func() {
		sigListener(ctx, sigs)
		cancel()
		wg.Done()
	}()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" { // за неимением less, пусть будет хоть какой то вывод
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
	l.Info("Приложение завершило работу штатно")
}
