package work

import (
	// "fmt"
	"github.com/garyburd/redigo/redis"
	"sort"
	"strconv"
	"strings"
)

type Client struct {
	namespace string // eg, "myapp-work"
	pool      *redis.Pool
}

func NewClient(namespace string, pool *redis.Pool) *Client {
	return &Client{
		namespace: namespace,
		pool:      pool,
	}
}

func (c *Client) WorkerPoolIDs() ([]string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	workerPoolsKey := redisKeyWorkerPools(c.namespace)

	vals, err := redis.Strings(conn.Do("SMEMBERS", workerPoolsKey))
	if err != nil {
		return nil, err
	}
	sort.Strings(vals)

	return vals, nil
}

// TODO: should we rename this heartbeat?
type WorkerPoolStatus struct {
	WorkerPoolID string
	StartedAt    int64
	HeartbeatAt  int64

	JobNames    []string
	Concurrency uint
	Host        string
	Pid         int

	WorkerIDs []string
}

func (c *Client) WorkerPoolStatuses(workerPoolIDs []string) ([]*WorkerPoolStatus, error) {
	conn := c.pool.Get()
	defer conn.Close()

	for _, wpid := range workerPoolIDs {
		key := redisKeyHeartbeat(c.namespace, wpid)
		conn.Send("HGETALL", key)
	}

	if err := conn.Flush(); err != nil {
		logError("worker_pool_statuses.flush", err)
		return nil, err
	}

	heartbeats := make([]*WorkerPoolStatus, 0, len(workerPoolIDs))

	for _, wpid := range workerPoolIDs {
		vals, err := redis.Strings(conn.Receive())
		if err != nil {
			logError("worker_pool_statuses.receive", err)
			return nil, err
		}

		heartbeat := &WorkerPoolStatus{
			WorkerPoolID: wpid,
		}

		for i := 0; i < len(vals)-1; i += 2 {
			key := vals[i]
			value := vals[i+1]

			var err error
			if key == "heartbeat_at" {
				heartbeat.HeartbeatAt, err = strconv.ParseInt(value, 10, 64)
			} else if key == "started_at" {
				heartbeat.StartedAt, err = strconv.ParseInt(value, 10, 64)
			} else if key == "job_names" {
				heartbeat.JobNames = strings.Split(value, ",")
				sort.Strings(heartbeat.JobNames)
			} else if key == "concurrency" {
				var vv uint64
				vv, err = strconv.ParseUint(value, 10, 0)
				heartbeat.Concurrency = uint(vv)
			} else if key == "host" {
				heartbeat.Host = value
			} else if key == "pid" {
				var vv int64
				vv, err = strconv.ParseInt(value, 10, 0)
				heartbeat.Pid = int(vv)
			} else if key == "worker_ids" {
				heartbeat.WorkerIDs = strings.Split(value, ",")
				sort.Strings(heartbeat.WorkerIDs)
			}
			if err != nil {
				logError("worker_pool_statuses.parse", err)
				return nil, err
			}
		}

		heartbeats = append(heartbeats, heartbeat)
	}

	return heartbeats, nil
}

type WorkerStatus struct {
	WorkerID string
	IsBusy   bool

	// If IsBusy:
	JobName   string
	JobID     string
	StartedAt int64
	ArgsJSON  string
	Checkin   string
	CheckinAt int64
}

// "job_name", obv.jobName,
// "job_id", obv.jobID,
// "started_at", obv.startedAt,
// "args", argsJSON,
// "checkin", obv.checkin,
// "checkin_at", obv.checkinAt,

func (c *Client) WorkerStatuses(workerIDs []string) ([]*WorkerStatus, error) {
	conn := c.pool.Get()
	defer conn.Close()

	for _, wid := range workerIDs {
		key := redisKeyWorkerStatus(c.namespace, wid)
		conn.Send("HGETALL", key)
	}

	if err := conn.Flush(); err != nil {
		logError("worker_statuses.flush", err)
		return nil, err
	}

	statuses := make([]*WorkerStatus, 0, len(workerIDs))

	for _, wid := range workerIDs {
		vals, err := redis.Strings(conn.Receive())
		if err != nil {
			logError("worker_statuses.receive", err)
			return nil, err
		}

		status := &WorkerStatus{
			WorkerID: wid,
		}

		for i := 0; i < len(vals)-1; i += 2 {
			key := vals[i]
			value := vals[i+1]

			status.IsBusy = true

			var err error
			if key == "job_name" {
				status.JobName = value
			} else if key == "job_id" {
				status.JobID = value
			} else if key == "started_at" {
				status.StartedAt, err = strconv.ParseInt(value, 10, 64)
			} else if key == "args" {
				status.ArgsJSON = value
			} else if key == "checkin" {
				status.Checkin = value
			} else if key == "checkin_at" {
				status.CheckinAt, err = strconv.ParseInt(value, 10, 64)
			}
			if err != nil {
				logError("worker_statuses.parse", err)
				return nil, err
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// // List jobs
// func (c *Client) Jobs() []string {
// 	// todo: how do we know this if we're not connected to a worker?
// 	// opt1: enqueue adds it to <ns>:jobs
// 	// opt2: we list keys on <ns>:jobs:* (using scan?)
// 	// opt3: we don't actually build this command. You configure it.
// 	// opt4: processing a job will add an entry to <ns>:jobs
//	// opt5: we base it on known workerpools and their jobs
// 	return nil
// }
//
// func (c *Client) JobCount(jobName string) int64 {
//
// }
//
// func (c *Client) JobLatency(jobName string) int64 {
//
// }
//
// func (c *Client) DeleteJobs(jobName string) {
// }
//
//
// type WorkerStatus struct {
// 	WorkerSetID string
// 	WorkerID string
//
// 	IsWorking bool
// 	JobName string
// 	StartedAt int64
// 	Checkin string
// }
//
// func (c *Client) WorkerStatuses(workerID []string) []*WorkerStatus {
//
// }