// Command portablezip packs one platform's build into the archive people run
// without installing: the executable and the resources beside it, at the
// archive's root, with the installers and the build's own bookkeeping left out.
//
//	go run ./tools/portablezip -dir build/windows-amd64 -out build/Fermata-1.0.0-windows-amd64-portable.zip
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("portablezip: ")

	dir := flag.String("dir", "", "the build directory of one platform, as build/windows-amd64")
	out := flag.String("out", "", "the archive to write, as build/Fermata-1.0.0-windows-amd64-portable.zip")
	flag.Parse()
	if *dir == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*dir, *out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(*out)
}

// run writes the archive of dir into out, and fails when dir holds nothing to
// ship: an archive of the wrong files is worse than a failed build.
func run(dir, out string) error {
	files, err := shippable(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("%s holds nothing to ship", dir)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, name := range files {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		// At the root, so that unzipping gives one folder of the app itself.
		h.Name = name
		h.Method = zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, src)
		src.Close()
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

// shippable lists the files of dir that belong in the portable archive. The
// installers are not among them: they install the app, which is the other way
// to run it.
func shippable(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Leave out the build's own bookkeeping, earlier archives, and the
		// installer, which is the other way to run the app.
		if strings.HasPrefix(name, ".") ||
			strings.HasSuffix(name, ".zip") ||
			strings.Contains(name, "-setup-") {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}
