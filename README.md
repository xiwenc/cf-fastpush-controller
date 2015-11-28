cf-fastpush-controller
==

This application starts a backend service during startup and exposes some REST actions that can be used to update files.

REST Actions
===

- `GET /files`: Returns a list a files of the current working directory during the launch of cf-fastpush-controller
- `PUT /files`: Update backend files. It automatically triggers a backend restart if a changed file matches `FASTPUSH_RESTART_REGEX`
- `POST /restart`: Triggers a backend restart
- `GET /status`: Get the controller and backend status

Configuration
===

Environment variables:
- `FASTPUSH_RESTART_REGEX`: automatically trigger a backend restart if a changed file path matches this regex
- `FASTPUSH_IGNORE_REGEX`: do not trigger a backend restart if a changed file path matches this regex


Related components
===

- cf-fastpush-plugin
- cf-mendix-buildpack (more to come)


Example usage
===

```
export FASTPUSH_RESTART_REGEX='^.*\.jar$'
export FASTPUSH_IGNORE_REGEX='^.*\.js$'
./cf-fastpush-controller "python -m http.server" "localhost:9000"
```

```
curl -i -X PUT http://localhost:9000/files -H 'Content-Type: application/json' -d '[{"Path": "somefile.jar", "Checksum": "deadbeef", "Content": "fMy4+UFqjADjVA=="}]'
```
