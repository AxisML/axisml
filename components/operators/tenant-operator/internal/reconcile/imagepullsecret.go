package reconcile

// imagepullsecret.go intentionally has no logic of its own; the
// ImagePullSecrets reconciler lives in secret.go where it shares the copy /
// drift / GC implementation with the generic Secret reconciler. This file
// is a placeholder to keep the package layout aligned with design §6.3 / §6.4.
