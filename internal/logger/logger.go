package logger

import "go.uber.org/zap"

// L is the shared SugaredLogger. It defaults to a no‑op logger to avoid nil
// dereference before the real logger is initialised in root.PersistentPreRunE.
var L = zap.NewNop().Sugar()
