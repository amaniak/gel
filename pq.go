package gel

import (
  "io"
  "os"
  "log"
  "bytes"
  "os/exec"
)


// pSQLWorker

func pSQLWorker(sql string) func(io.WriteCloser) {
  return func(stdin io.WriteCloser) {
    io.Copy(stdin, bytes.NewBufferString(sql))
    io.Copy(stdin, bytes.NewBufferString("\\q\r"))
    stdin.Close()
  }
}

// PostgrSQL Commander

func pSQLCommander(processPSQLWorker func(io.WriteCloser)) {



  // build command
  args := []string{"-v", "ON_ERROR_STOP=1", "-d", "ensure"}
  cmd := exec.Command("psql", args...)

  // pipe out
  stdout, err := cmd.StdoutPipe()
  if err != nil {
    log.Printf("Error pipe stdout")
    log.Panic(err)
  }

  // pipe in
  stdin, err := cmd.StdinPipe()
  if err != nil {
    log.Printf("Error pipe stdin")
    log.Panic(err)
  }

  stderr, err := cmd.StderrPipe()
  if err != nil {
    log.Printf("Error pipe stderr")
    log.Panic(err)
  }

  // defers
  defer stdout.Close()
  defer stderr.Close()
  defer stdin.Close()

  // pipe to STDIN
  go processPSQLWorker(stdin);

  // pipe to STDOUT
  go io.Copy(os.Stdout, stdout)

  // pipe to STDERR
  go io.Copy(os.Stderr, stderr)

  // run
  err = cmd.Start()

  // panic
  if err != nil {
    log.Printf("Error start")
    log.Panic(err)
  }

  // wait
  err = cmd.Wait()


  // trace error
  if(err != nil){
    log.Printf("Error wait")
    log.Fatal(err)
  }
}

// PostgrSQL Dump (RUN here)

func PQDump(sql string) {
  pSQLCommander(pSQLWorker(sql))
}
