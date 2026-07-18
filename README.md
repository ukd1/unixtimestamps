# Unix timestamps as-a-service

This is a stupid hack project for me to learn some more go, especially figuring out the streaming parts of GIN. You can view it online at https://unixtimestamps.rsmith.co/.

## Deployment

Container builds publish a commit-addressed tag and attach a Kubernetes manifest pinned to the image digest. To render the manifest manually, set `IMAGE_DIGEST` to a `sha256:...` digest and run:

```sh
sed "s|\${IMAGE_DIGEST}|${IMAGE_DIGEST}|g" k8s.yml | kubectl apply -f -
```
