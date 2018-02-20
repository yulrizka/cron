# CRON
[![GoDoc](http://godoc.org/github.com/yulrizka/cron?status.png)](http://godoc.org/github.com/yulrizka/cron) 
[![Build Status](https://travis-ci.org/yulrizka/cron.svg?branch=master)](https://travis-ci.org/yulrizka/cron) 
[![Coverage Status](https://coveralls.io/repos/github/yulrizka/cron/badge.svg?branch=master)](https://coveralls.io/github/yulrizka/cron?branch=master)

Go library that schedule and parses cron expression

Features:
* Simple parser for cron expression.
* Using SQLStore, it allows to run multiple instance of the application for high availability without needing
  external tool to do leader election (consul, zookeeper, etc).
  It utilizes SQL **LOCK** tables functionality.
* During initialization fo SQLStore, it will make sure that the tables exists.

## License
MIT License Copyright (c) 2018 Ahmy Yulrizka

## CRON Expression
```
  +------------------ Minute (0-59)       : [5]
  | +---------------- Hour (0-23)         : [0, 1, 2, ..., 23]
  | |   +------------ Day of month (1-31) : [5, 10, 15, 20, 30]
  | |   |    +------- Month (1-12)        : [1, 3, 5, ..., 11]
  | |   |    |     +- Day of Week  (0-6)  : [Sun, Mon, Tue, Wed]
  5 *  */5 1-12/2 0-3
```

## Example
```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/yulrizka/cron"
)

func main() {
	ctx := context.Background()

	// log error that can't be returned as return value. It let you choose how you would log the errors
	// if you don't read from the channel, error will be silently discarded.
	go func() {
		for err := range cron.ErrorCh {
			log.Printf("[ERROR][CRON] %v", err)
		}
	}()

	// this create cron entry by parsing expression.
	expression := "* * * * *"
	location := time.UTC
	name := "ENTRY_1"
	entry, err := cron.Parse(expression, location, name)
	if err != nil {
		log.Fatal(err)
	}

	// if you want to check whether a time matches with the expression
	if entry.Match(time.Now()) {
		// cron entry matched
	}

	// MemStore implements Store object which persist entries and events (triggered entries) to memory store.
	// This store is volatile and used here for example. 
	// For real usage use persisted SQLStore (other example below).
	store := &cron.MemStore{}
	store.AddEntry(ctx, entry)

	// handler function that will be called by the scheduler if an entry is triggered
	handler := func(name string) {
		// filter the by the job name
		switch name {
		case "ENTRY_1":
			log.Printf("handling job %q", name)
		default:
			log.Printf("[ERROR] unknown job %q", name)
		}
	}

	// setup the scheduler
	scheduler := cron.NewScheduler(handler, store)
	if err := scheduler.Run(ctx); err != nil {
		log.Printf("[ERROR] scheduler found error: %v", err)
	}
}


```

**with SQLStore**

Use SQLStore as backend for persistent. This should be use in most (production) cases.
By using this store, you can run more than one instances of the application (for high availability).
It will make sure that the job will be executed only once by using SQL lock tables.

```go
package main

import (
	"context"
	"database/sql"
	"log"
	
	_ "github.com/go-sql-driver/mysql" // use the latest mysql driver

	"github.com/yulrizka/cron"
)

func main() {
	// mysql > INSERT INTO _entries (expression, location, name) VALUES ("* * * * *", "UTC", "ENTRY_1")

	ctx := context.Background()
	db, err := sql.Open("mysql", "username:password@tcp(127.0.0.1:3306)/cron")
	if err != nil {
		log.Fatal(err)
	}

	sqlStore, err := cron.NewSQLStore(db)
	if err != nil {
		log.Fatalf("Failed to initialize MysqlPersister: %v", err)
	}

	// handler function that will be called
	handler := func(name string) {
		switch name {
		case "ENTRY_1":
			log.Printf("handling job %q", name)
		default:
			log.Printf("[ERROR] unknown job %q", name)
		}
	}

	// start the scheduler with handler above
	scheduler := cron.NewScheduler(handler, sqlStore)
	if err != nil {
		log.Fatalf("failed to initialize scheduler: %v", err)
	}

	// Run will check existance of required tables. If not found, it will try to create it
	if err := scheduler.Run(ctx); err != nil {
		log.Fatalf("scheduler does not terminate properly: %v", err)
	}
}
```


## Limitation
Current limitation (by design)

* Does not take account daylight saving time.

  example during CEST -> CET, 02:00:00 will be run twice (at 00:00:00 UTC (02:00 CEST) and 01:00:00 UTC (02:00 CET)).

  If possible always use UTC. It does mean that on DST task will be executed 1 hour earlier. The alternative
  would be to schedule the job one minute early or after (ex: 01:59:00 or 03:01:00).

* Each run execution is in separate go routine, which mean if your job takes more than one minute to execute
  next one will fire one minute after the first one. It will not wait until your first job finished.

  If you want to skip second job (on the next minute), you have to handle this with semaphore (buffered channel).

  ```go
  semA := make(chan struct{}, 1)

  func handler(entry cron.Entry) {
    if entry.Name == "JOB A" {
        select {
        case semA <- struct{}{}:
            // run your stuff here
            <-semA
        default:
           // already process, you can skip or block
        }
    }
  }
  ```

* For Simplicity  macros (@yearly, @monthly, @daily, ...) are not supported. This can easily be expressed by normal
  expression.
  
## Projects using this lib
* TODO

Let me know if you want to list your project here.  

## Contributions
Contribution are always welcome. Please create a github issue first and describe your suggestion/plan so that we can
have open discussion about it.

If you have a bug or issue, please also post it as github issue.

## Attributions
Thanks to
* Rob Figueiredo for (https://github.com/robfig/cron) which give inspiration for the parser.
