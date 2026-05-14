# RedisForge Docs

This folder is the RedisForge learning path. Use it when you want to revise Redis concepts, understand the implementation, or decide what to build next.

## Recommended Reading Order

1. [Implementation Architecture](implementation/architecture.md)
   Start here to understand how the HTTP API, repositories, Redis wrappers, workers, metrics, and config fit together.

2. [Redis Patterns](implementation/redis-patterns.md)
   Read this when you want the Redis-specific lessons: JSON, Bloom, Search, Streams, Pub/Sub, Sentinel, and Cluster.

3. [Build Guide](REDISFORGE_BUILD_GUIDE.md)
   Historical phase-by-phase record of how the project was built.

4. [Redis Decisions](redis-decisions.md)
   Deeper notes on configuration choices and tradeoffs.

5. [Profiling Results](profiling-results.md)
   Performance and tuning workspace for SLOWLOG, LATENCY, MEMORY, and benchmarks.

6. [Demo Workflow](demo_workflow.md)
   A short repeatable flow for recording demos and proving the project works.

7. [Project Journal](project-journal.md)
   Commit cadence, next public updates, and the habit loop that keeps the repo looking alive.

## Docs By Intent

| Intent | Read |
| --- | --- |
| "I forgot how Redis Streams consumer groups work." | [implementation/redis-patterns.md](implementation/redis-patterns.md) |
| "I want to understand the codebase fast." | [implementation/architecture.md](implementation/architecture.md) |
| "I want to know why these Redis settings exist." | [redis-decisions.md](redis-decisions.md) |
| "I want to record a demo for GitHub/LinkedIn/X." | [demo_workflow.md](demo_workflow.md) and [social_templates.md](social_templates.md) |
| "I want to keep making real commits." | [project-journal.md](project-journal.md) |
| "I am preparing for Redis/backend interviews." | [interview_questions.md](interview_questions.md) and [redisforge_interview_answers.md](redisforge_interview_answers.md) |

## Documentation Standard

Every meaningful feature should leave behind three things:

- Code that demonstrates the behavior.
- A test, benchmark, or manual verification step.
- A doc note explaining why the Redis choice was made.

This keeps RedisForge useful for both the maintainer and future visitors.

