package runner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	log "chill/util"

	"github.com/fsnotify/fsnotify"
)

func watch(path string, abort chan struct{}) (<-chan string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	out := make(chan string)
	go func() {
		defer close(out)
		defer watcher.Close()

		for {
			select {
			case <-abort:
				// Abort watching
				err := watcher.Close()
				if err != nil {
					log.Error("Failed to stop watch")
				}
				return
			case fp := <-watcher.Events:
				if fp.Op == fsnotify.Create {
					info, err := os.Stat(fp.Name)
					if err == nil && info.IsDir() {
						// Add newly created sub directories to watch list
						log.Trace("Add newly diectory ( %s )", fp.Name)
						watcher.Add(fp.Name)
					}
				}

				if fp.Op&fsnotify.Write == fsnotify.Write {
					out <- fp.Name
				}

			case err := <-watcher.Errors:
				log.Error("watch error: %s", err.Error())
			}
		}
	}()

	// Start watch
	{
		var paths []string
		currpath, _ := os.Getwd()

		readAppDirectories(currpath, &paths)

		log.Info("Start watching......")

		for _, dir := range paths {
			watcher.Add(dir)
			log.Trace("Directory( %s )", dir)
		}
	}

	return out, nil
}

func match(in <-chan string, patterns []string) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)
		for fp := range in {
			info, err := os.Stat(fp)

			if os.IsNotExist(err) {
				log.Trace("Dictory (%s) have been removed", fp)
				continue
			}

			// here have bug
			if os.IsExist(err) || !info.IsDir() {
				//Split splits path immediately following the final Separator,
				//separating it into a directory and file name component.
				//If there is no Separator in path,
				//Split returns an empty dir and file set to path.
				//The returned values have the property that path = dir+file.
				_, fn := filepath.Split(fp)
				for _, p := range patterns {
					if ok, _ := filepath.Match(p, fn); ok {
						out <- fp
					}
				}
			}
		}
	}()
	return out
}

func gather(first string, changes <-chan string, delay time.Duration) []string {
	files := make(map[string]bool)
	files[first] = true
loop:
	for {
		select {
		case fp := <-changes:
			files[fp] = true
		case <-time.After(delay):
			break loop
		}
	}

	ret := []string{}
	for value := range files {
		ret = append(ret, value)
	}

	sort.Strings(ret)
	return ret
}

func readAppDirectories(directory string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(directory)

	if err != nil {
		return
	}

	haveDir := false
	for _, fileinfo := range fileInfos {
		if fileinfo.IsDir() == true && fileinfo.Name() != "." && fileinfo.Name() != ".git" {
			readAppDirectories(directory+"/"+fileinfo.Name(), paths)
			continue
		}

		if haveDir {
			continue
		}

		if filepath.Ext(fileinfo.Name()) == ".go" {
			*paths = append(*paths, directory)
			haveDir = true
		}
	}
}

// var watchExts = []string{""}

// func checkIfWatchModify(name string) bool {
// 	for _, s := range watchExts {
// 		if strings.HasSuffix(name, s) {
// 			return true
// 		}
// 	}
// 	return false
// }
