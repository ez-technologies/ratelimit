# ratelimit

An in-memory Go rate limiter with one token bucket per key (built on `golang.org/x/time/rate`).

Idle keys are forgotten only once their bucket would be full again, so flooding the limiter with new keys cannot
reset a limit that is in use. Limits are per process.

## Install

```sh
go get github.com/ez-technologies/ratelimit
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/ez-technologies/ratelimit"
)

func main() {
	l := ratelimit.New(10 * time.Minute) // forget keys idle for 10 minutes (or a full refill, if longer)

	// Each user: a burst of 5, then one more every second.
	for i := 0; i < 7; i++ {
		fmt.Println(l.Allow("user:42", time.Second, 5))
	}
}
```

The first call for a key fixes its rate; later calls for the same key with other values use the first ones.

## Licence

MIT, see [LICENSE](LICENSE).
