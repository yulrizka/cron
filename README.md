# CRON

Go library that schedule and parses cron expression

Doc: TODO

Features:
* Simple parser for cron expression.
* Using SQLStore, it allows to run multiple instance of the application for high availability without needing
  external tool to do leader election (consul, zookeeper, etc).
  It require the ability of the SQL database to do **LOCK** on tables.
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
### Using the scheduler
create a job handler that will be called when the entry is triggered
```go
handler := func(e cron.Entry) {
    switch e.Name {
    case "JOB A":
        log.Println("Handling JOB A")
    default:

    }
}

```

**with MySQL store**
```go
// mysql > INSERT INTO _entries (expression, name) VALUES ("* * * * *", "JOB A")

db, err := sql.Open("mysql", "username:password@tcp(127.0.0.1:3306)/cron")
if err != nil {
    log.Fatal(err)
}

persister, err := cron.NewSQLStore(db) // This will check and create necessary table if not exists
if err != nil {
    log.Fatalf("Failed to initialize MysqlPersister: %v", err)
}

// start the scheduler with handler above
scheduler, err := cron.NewScheduler(context.Backgroun(), handler, persister)
if err != nil {
    log.Fatalf("failed to initialize scheduler: %v", err)
}

if err := scheduler.Run(); err != nil {
    log.Fatalf("scheduler does not terminate properly: %v". err)
}
```

**With in memory (volatile) store**
```go
memPersister := cron.NewMemoryStore()

entry, err := cron.parse("5 *  */5 1-12/2 0-3", "JOB A", time.UTC)
if err != nil {
    return fmt.Errorf("failed to parse entry: %v", err)
}
memPersister.Add(entry)

scheduler, err := cron.NewScheduler(context.Backgroun(), handler, mempersister)
... // same as above
```

### using only the parser
If you just need ability to parse and check cron expression
```go
entry, err := cron.parse("5 *  */5 1-12/2 0-3", "JOB A", time.UTC)
if err != nil {
    return fmt.Errorf("failed to parse entry: %v", err)
}
t := time.Date(2006, 1, 2, 15, 4, 0, 0, time.UTC), // Monday, 2 January 2006 15:04:00 UTC
if entry.Match(t) {
    // t matched the cron expression
}
```

## Limitation
Current limitation (by design)

* Does not take account daylight saving time.

  example during CEST -> CET, 02:00:00 will be run twice (at 00:00:00 UTC (02:00 CEST) and 01:00:00 (02:00 CET)).

  If possible always use UTC. It does mean that on DST task will be executed 1 hour earlier. The alternative
  is to schedule the job one minute early or after (ex: 01:59:00 or 03:01:00).

* Each run execution is in separate go routine. Which mean if your job takes more than one minute to execute,
  next one will fire one minute after the first one.

  If you want to skip job that next minute, you have to
  handle this with semaphore (buffered channel).

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

## Contribute
Contribution are always welcome. Please create a github issue first and describe your suggestion/plan to have a discussion.

If you have a bug or issue, please also post it on github issue

## Attribution
Thanks to
* Rob Figueiredo for (https://github.com/robfig/cron) which give inspiration for the parser.
