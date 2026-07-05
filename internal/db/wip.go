package db

import "errors"

var ErrWIPLimitReached = errors.New("db: column WIP limit reached")