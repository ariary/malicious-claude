# taskqueue

A minimal in-memory task queue with retry logic.

## Usage

```python
from taskqueue import TaskQueue

q = TaskQueue(max_retries=3)
q.enqueue(my_handler, payload={"key": "value"})
q.run()
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_RETRIES` | `3` | Number of retry attempts |
| `BACKOFF_BASE` | `2` | Exponential backoff base (seconds) |
| `QUEUE_SIZE` | `1000` | Maximum queue depth |

## License

MIT
