package EasyLogger

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultLogDir      = "./Logs/"
	FileNameTimeFormat = "2006-01-02"
	FileNameExt        = ".log"
	CompressSuffix     = ".gz"

	NanosecondPerDay = 24 * 3600 * time.Second
)

var (
	osStat = os.Stat
)

// ensure we always implement io.WriteCloser
var _ io.WriteCloser = (*Logger)(nil)

// this aims to have a time rotating logger depending on days

type Logger struct {
	// Directory is the place to store log files.
	// Default is "./Logs/"
	Directory string

	// MaxDays is the maximum number of days to rotate.
	// The default is not rotating.
	MaxDays int

	// MaxBackups is the maximum number of files to retain.
	// The default is to retain all old files.
	MaxBackups int

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool

	currentFile *os.File
	mu          sync.Mutex
	millCh      chan bool
	startMill   sync.Once
}

// Close implements io.Closer, and closes the current logfile.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.close()
}

// close method closes the currentFile if it is open.
func (l *Logger) close() error {
	if l.currentFile == nil {
		return nil
	}
	err := l.currentFile.Close()
	l.currentFile = nil
	return err
}

func (l *Logger) dir() string {
	//parent := os.Args[0]
	fi, err := osStat(l.Directory)
	if err != nil {
		if os.IsNotExist(err) {
			err2 := os.MkdirAll(l.Directory, 0766)
			if err2 != nil {
				l.Directory = DefaultLogDir
			}
		}
	} else {
		if !fi.IsDir() {
			l.Directory = DefaultLogDir
		}
	}
	return l.Directory
}

// mill performs post-rotation compression and removal of stale log files,
// starting the mill goroutine if necessary.
func (l *Logger) mill() {
	l.startMill.Do(func() {
		l.millCh = make(chan bool, 1)
		go l.millRun()
	})
	select {
	case l.millCh <- true:
	default:
	}
}

// millRun runs in a goroutine to manage post-rotation compression and removal
// of old log files.
func (l *Logger) millRun() {
	for range l.millCh {
		// what am I going to do, log this?
		_ = l.millRunOnce()
	}
}

// millRunOnce performs compression and removal of stale log files.
// Log files are compressed if enabled via configuration and old log
// files are removed, keeping at most l.MaxBackups files, as long as
// none of them are older than MaxAge.
func (l *Logger) millRunOnce() error {
	if l.MaxBackups == 0 && !l.Compress {
		return nil
	}

	files, err := l.oldLogFiles() // will get all log files including the latest writing one
	if err != nil {
		return err
	}

	var compress, remove []logInfo

	if l.MaxBackups > 0 && l.MaxBackups < len(files) {
		preserved := make(map[string]bool)
		var remaining []logInfo
		for _, f := range files {
			// Only count the uncompressed log file or the
			// compressed log file, not both.
			fn := f.Name()
			if strings.HasSuffix(fn, CompressSuffix) {
				fn = fn[:len(fn)-len(CompressSuffix)]
			}
			preserved[fn] = true

			if len(preserved) > l.MaxBackups {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	if l.Compress {
		temp := files[1:]
		for _, f := range temp {
			if !strings.HasSuffix(f.Name(), CompressSuffix) {
				compress = append(compress, f)
			}
		}
	}

	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(l.dir(), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}
	for _, f := range compress {
		fn := filepath.Join(l.dir(), f.Name())
		errCompress := compressLogFile(fn, fn+CompressSuffix)
		if err == nil && errCompress != nil {
			err = errCompress
		}
	}

	return err
}

// newFileName creates a new file name
func (l *Logger) newFileName() string {
	t := time.Now()
	if !l.LocalTime {
		t = t.UTC()
	}
	currentDate := t.Format(FileNameTimeFormat)
	name := currentDate + FileNameExt
	return filepath.Join(l.dir(), name)
}

// oldLogFiles returns the list of all log files stored in the same
// directory as the current log currentFile, sorted by time stamp in currentFile name
func (l *Logger) oldLogFiles() ([]logInfo, error) {
	files, err := ioutil.ReadDir(l.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log currentFile directory: %s", err)
	}
	logFiles := []logInfo{}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if t, err := l.timeFromName(f.Name(), FileNameExt); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		if t, err := l.timeFromName(f.Name(), FileNameExt+CompressSuffix); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		// error parsing means that the suffix at the end was not generated
		// by lumberjack, and therefore it's not a backup currentFile.
	}

	// sort by date descending
	sort.Sort(sort.Reverse(byFormatTime(logFiles)))

	return logFiles, nil
}

// openNew opens a new log currentFile for writing, moving any old log currentFile out of the
// way.  This methods assumes the currentFile has already been closed.
func (l *Logger) openNew() error {
	newFileName := l.newFileName()

	// we use truncate here because this should only get called when we've moved
	// the currentFile ourselves. if someone else creates the currentFile in the meantime,
	// just wipe out the contents.
	f, err := os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}
	l.currentFile = f
	l.mill()
	return nil
}

// openExistingOrNew opens the logfile if its timestamp is in the log interval.
// If there is no such currentFile, a new currentFile is created.
func (l *Logger) openExistingOrNew() error {
	allFiles, err := l.oldLogFiles()
	if err != nil {
		return err
	}
	if len(allFiles) > 0 {
		latest := allFiles[0]
		t := time.Now()
		if !l.LocalTime {
			t = t.UTC()
		}
		duration := t.Sub(latest.timestamp)
		if duration < time.Duration(l.MaxDays)*NanosecondPerDay {
			// use the latest file to log
			file, err := os.OpenFile(l.dir()+latest.Name(), os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			l.currentFile = file
			return nil
		}
	}
	// create a new file
	return l.openNew()
}

// timeFromName extracts the formatted time from the filename by stripping off
// the filename's prefix and extension. This prevents someone's filename from
// confusing time.parse.
func (l *Logger) timeFromName(filename string, ext string) (time.Time, error) {
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errors.New("mismatched extension")
	}
	ts := filename[:len(filename)-len(ext)]
	return time.Parse(FileNameTimeFormat, ts)
}

func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile == nil {
		if err = l.openExistingOrNew(); err != nil {
			return 0, err
		}
	}
	n, err = l.currentFile.Write(p)
	return n, err
}

// compressLogFile compresses the given log file, removing the
// uncompressed log file if successful.
func compressLogFile(src, dst string) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	fi, err := osStat(src)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	if err := chown(dst, fi); err != nil {
		return fmt.Errorf("failed to chown compressed log file: %v", err)
	}

	// If this file already exists, we presume it was created by
	// a previous attempt to compress the log file.
	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		return fmt.Errorf("failed to open compressed log file: %v", err)
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)

	defer func() {
		if err != nil {
			os.Remove(dst)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()

	if _, err := io.Copy(gz, f); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := gzf.Close(); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return err
	}

	return nil
}

// logInfo is a convenience struct to return the filename and its embedded
// timestamp.
type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.Before(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
