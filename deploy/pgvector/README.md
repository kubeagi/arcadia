# build pgvector image based on bitnami/postgresql

`bitnami/postgresql` provides easy to use and proven postgresql images and helm charts, but this version does not provide the `pgvector` extension. So we built the pgvector image on top of that. This means that all the helm chart parameters can still be used and the out-of-the-box `pgvector` can be used at the same time.

## how all things work

Except for this `README.md` and `run.sh`, all other files in the current directory come from the [bitnami/postgresql](https://github.com/bitnami/containers) project, which we need to keep in git to build a reproducible `pgvector` image. The build is done using the `GitHub Action` workflow, see file `.github/workflows/pgvector_image_build.yml` for details. The image is automatically built when merging code when the current directory changes. **So we only need to update the current directory to update the `pgvector` image.**

## how to use

just run `run.sh` file, it will:

1. remove old files
2. get base dockerfile and script from bitnami
3. add pgvector build script to Dockerfile

Also, you may need update `env.TAG` in line 12 in `.github/workflows/pgvector_image_build.yml`,it specifies the tag information for the built image, e.g.

```yaml
env:
    TAG: 16.1.0-debian-11-r18-pgvector-v0.5.1
```

means the pgvector image will be named `kubeagi/postgresql:16.1.0-debian-11-r18-pgvector-v0.5.1`

You can then pull request, wait for it to merge, and the GitHub action will work.
I don't recommend building this image locally, it only takes 1 minute to build an amd64 image on GitHub, and 10 minutes to build a dual-architecture image of amd64 and arm64, but fixing the network issues locally will take a long time, and odds are there will be multiple places where the network issues are triggered.
