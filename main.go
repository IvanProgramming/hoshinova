package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/HoloArchivists/hoshinova/config"
	"github.com/HoloArchivists/hoshinova/logger"
	"github.com/HoloArchivists/hoshinova/notifier"
	"github.com/HoloArchivists/hoshinova/recorder"
	"github.com/HoloArchivists/hoshinova/taskman"
	"github.com/HoloArchivists/hoshinova/uploader"
	"github.com/HoloArchivists/hoshinova/util"
	"github.com/HoloArchivists/hoshinova/watcher"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	lg := logger.New(cfg.Logging.Level)
	tm := taskman.New(cfg, lg)

	lg.Info("Using log level", cfg.Logging.Level)
	lg.Info("Starting hoshinova")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	ctx = util.WithLogger(ctx, lg)
	ctx = util.WithConfig(ctx, cfg)
	ctx = util.WithTaskManager(ctx, tm)

	var uploaders []uploader.Uploader
	var notifiers []notifier.Notifier

	// Create uploaders
	for _, u := range cfg.Uploaders {
		upl, err := uploader.NewUploader(u)
		if err != nil {
			panic(err)
		}
		uploaders = append(uploaders, upl)
	}

	// Create notifiers
	for _, n := range cfg.Notifiers {
		not, err := notifier.NewNotifier(n)
		if err != nil {
			panic(err)
		}
		notifiers = append(notifiers, not)
	}

	// Start watching the channels for new videos
	wg := watcher.Watch(ctx)

	// Subscribe to events
	go func() {
		ch := tm.Subscribe(taskman.StepIdle)

		for {
			select {
			case <-ctx.Done():
				return
			case task := <-ch:
				// Create a new goroutine for each task
				go func(task *taskman.Task) {
					rec, err := recorder.RecordVideo(ctx, task)
					if err != nil {
						tm.UpdateStep(task.Video.Id, taskman.StepErrored)
						return
					}
					lg.Debugf("(%s) Recording finished: %#v\n", task.Video.Id, rec)

					for i, upl := range uploaders {
						lg.Info("Uploading", task.Video.Id, "to", cfg.Uploaders[i].Name)
						res, err := upl.Upload(ctx, rec)
						if err != nil {
							lg.Error("Error uploading", task.Video.Id, err)
							tm.UpdateStep(task.Video.Id, taskman.StepErrored)
							return
						}
						lg.Debug("Uploaded", task.Video.Id, res)

						for j, not := range notifiers {
							lg.Debug("Notifying", task.Video.Id, "with", cfg.Notifiers[j].Name)
							if err := not.NotifyUploaded(ctx, res); err != nil {
								lg.Error("Error notifying", task.Video.Id, err)
								tm.UpdateStep(task.Video.Id, taskman.StepErrored)
								return
							}
						}
						tm.UpdateStep(task.Video.Id, taskman.StepDone)
					}
				}(task)
			}
		}
	}()

	// Handle interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Print table when the program is interrupted
	lg.Info("Application started. Press Ctrl+C once to print the status, twice to exit.")
	func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-c:
				fmt.Println("")
				tm.PrintTable()
				lg.Info("Press Ctrl+C again to exit")

				select {
				case <-c:
					return
				case <-time.After(time.Second):
				}
			}
		}
	}()
	lg.Debug("Cancelling the context")
	cancel()

	lg.Info("Waiting for all goroutines to finish...")
	lg.Info("Press Ctrl+C again to force exit")
	go func() {
		<-c
		lg.Info("Force exiting")
		os.Exit(1)
	}()
	wg.Wait()
}
