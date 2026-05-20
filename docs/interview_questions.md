# interview_questions_general_project.md

**Total Questions: 52**

## 1. High-Level Project Understanding & Product Vision
1. Walk me through what this project actually does, end-to-end, from a user’s perspective.
2. Who is the target user? How would you pitch the value of this system to them in 30 seconds?
3. What core problem does this solve that a simpler manual workflow or an existing off-the-shelf tool doesn’t?
4. How do you measure the success, performance, or adoption of this tool? What metrics prove it's working as intended?
5. Explain the domain boundaries. What features or capabilities fall strictly *outside* the scope of this project, and why did you draw the line there?

## 2. Architecture & Overall Design
6. Draw (in text or on a whiteboard) the high-level architecture of your project — major components, services, and the request/data lifecycle.
7. Why did you choose this specific architectural pattern (e.g., Monolith, Microservices, SPA, Server-Rendered) over the alternatives? What were the trade-offs?
8. How do you handle dependency injection, state management, and configuration wiring across the application?
9. Explain your request lifecycle or middleware pipeline. What intercepts a request before it hits your core business logic, and in what order?
10. How is contextual data (like user sessions, localization, or request IDs) passed safely through the application layers?
11. If you had to extract your core domain logic and attach it to a completely different transport layer (e.g., swapping HTTP for a CLI or gRPC), how painful would it be? 

## 3. API Design & Routing Layer
12. Why did you choose your specific web framework or routing library over the standard library or more opinionated alternatives?
13. How do you handle cross-site vulnerabilities (CSRF/XSS), payload validation, and general API security?
14. How does your application handle potentially malicious or oversized request payloads? Is there a risk of memory exhaustion?
15. Explain your approach to input validation. Where does it live (handler vs. domain model), and how are validation errors propagated back to the client seamlessly?
16. How did you design your API contracts? Are you using REST, GraphQL, RPC? Why?

## 4. Database Layer (The Real Meat)
17. Explain your database stack. Why did you choose this specific datastore and ORM/query builder over the alternatives?
18. Walk me through how a complex write operation flows through the data layer, specifically detailing how you handle database transactions.
19. Describe your core database schema. How did you model the most complex relationship or entity hierarchy?
20. How do you ensure atomicity between multiple dependent state changes (e.g., updating a record and writing an audit log)?
21. What specific indexes, foreign keys, or constraints did you add? Is there any complex query you optimized that you’re proud of?
22. How does connection pooling work in your application under load? What metrics would you monitor to tune the pool size?
23. What transaction isolation level are you using for your most critical workflows? Could a race condition result in a lost update or phantom read?
24. How are you handling database migrations? Describe the CI/CD flow for applying a schema change safely to a production environment without downtime.
25. Explain your strategy for pagination on your largest tables as they grow to millions of rows (Offset/Limit vs. Cursor/Keyset).

## 5. Sessions, Auth & Security
26. How is session management or token authentication implemented (e.g., JWTs, Redis-backed sessions, database sessions)? What are the trade-offs of your choice?
27. Explain the full authentication flow from client request to database verification.
28. How do you handle Role-Based Access Control (RBAC) or granular permissions? Is it enforced in middleware, or do you check inside domain handlers?
29. How do you protect against session hijacking, token theft, and fixation? 
30. Walk me through how secrets or passwords are hashed and verified. What algorithm are you using, and how are you preventing timing attacks?
31. True/False: “Your current auth setup is sufficient for a public, enterprise-level production release.” Justify your answer.

## 6. Concurrency, Background Jobs & Error Handling
32. Does this project utilize multithreading, asynchronous processing, or background workers? If yes, how do you prevent resource leaks and race conditions?
33. How does graceful shutdown work in this app? How do you ensure active connections or processes finish before the server dies?
34. Walk me through your error-handling strategy across the app. How do you ensure you don't leak sensitive stack traces to the client while keeping logs useful for debugging?
35. Walk me through context/cancellation propagation. If a user cancels a request (e.g., closes their browser tab) during a slow database query, does the DB query cancel?
36. Are there any background workers or cron jobs? How do you ensure idempotent execution if you scale the application horizontally (so jobs don't run twice)?

## 7. Testing & Observability
37. What is your testing philosophy for this project? What is your ratio of unit vs. integration vs. end-to-end tests?
38. How do you test core business logic that depends on the database or third-party APIs? (Mocking vs. Stubs vs. Fakes).
39. Have you considered using ephemeral databases (like Testcontainers) for true integration testing? Why or why not?
40. What is your strategy for detecting concurrency bugs, data races, or flaky tests in your test suite?
41. How do you manage test data creation and fixture setup to keep tests isolated and deterministic?

## 8. Production Readiness & Scale
42. If this application is currently running on a single instance, what specific code or infrastructure changes are required to run it horizontally behind a load balancer?
43. Rate limiting, circuit breaking, or back-pressure — where would you add them in this architecture, and why?
44. Prioritize the top 3 engineering tasks you would need to ship *next* to make this ready for a massive public launch.
45. Describe your logging format. Is it structured for log aggregation systems? How do you trace a single user's request across multiple logs or services?
46. Imagine an unexpected 100x traffic spike. What infrastructure, memory, or database bottleneck in your system would break first, and how would you mitigate it?

## 9. Trade-offs, Alternatives & Senior-Level Thinking
47. Why did you choose your primary datastore (e.g., Relational SQL) over the alternatives (e.g., NoSQL, Graph, or Embedded DBs) for this specific domain?
48. Compare your approach of building this with your chosen stack versus using a heavy "batteries-included" framework or a low-code SaaS. Why was your approach worth the engineering effort?
49. If you had to rebuild the core feature of this project today, what would you change architecturally based on what you know now?
50. What was the single biggest architectural, performance, or language-specific lesson you learned while building this project?
51. If I asked you to add real-time, live-updating features (e.g., WebSockets, SSE) to the core entity, how would it change your architecture?
52. If this system becomes mission-critical and cannot go down, how would you design for high availability and disaster recovery?