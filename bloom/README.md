# Bloom

Bloom is a tool for start/stopping successfully pushed application that have been
pushed by [cedar](https://github.com/cloudfoundry/diego-stress-tests/tree/master/cedar).

## Usage

```bash
bloom -k <concurrency> -i <path-to-cedar-receipt> (--stop)
```

### -k <concurrency>
An integer flag that specified how many start/stop operations to do concurrently.

### -i <path-to-cedar-receipt>
The path to the cedar receipt file.

### (--stop)
Optional boolean flag to enable stop mode.
