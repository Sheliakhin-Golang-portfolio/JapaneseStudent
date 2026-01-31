# JapaneseStudent

A backend-focused Go web application for learning Japanese alphabets, rules and vocabulary, built to demonstrate production-oriented backend architecture, asynchronous processing, and service separation.

---

## What This Project Demonstrates

This project was created as a portfolio application with a focus on backend engineering practices rather than UI polish.

It demonstrates:

- Designing backend systems beyond simple CRUD or authentication
- Clear separation of responsibilities across services
- Background and scheduled task processing using durable queues
- Use of a relational database (MariaDB) as a source of truth
- Reliable execution, retries, and failure handling
- Maintainable and testable Go code

---

## Product Overview

JapaneseStudent provides functionality for learning Japanese alphabets (Hiragana, Katakana) and related exercises through tests, repeating vocabulary and attending lessons.

The learning domain is intentionally familiar — the primary goal of the project is to demonstrate backend system design and execution flows.

---

## Inspiration & Acknowledgements

This project was inspired by tools and resources that helped me personally while learning Japanese.

It is **not intended to compete** with these products, but rather to apply backend engineering practices to a familiar learning domain.

- **JapanesePod101**  
  https://www.japanesepod101.com

- **Katakana Memory Hint**  
  https://play.google.com/store/apps/details?id=jp.jfkc.KatakanaMemoryHintApp.En

- **Hiragana Memory Hint**  
  https://play.google.com/store/apps/details?id=jp.jfkc.HiraganaMemoryHintApp.En

- **Anki**  
  https://apps.ankiweb.net

---

## Architecture Overview

At a high level:

- HTTP APIs handle user requests
- MariaDB stores all persistent application state
- Redis is used for asynchronous and scheduled background tasks
- Workers process tasks and persist execution results

MariaDB acts as the **source of truth**, while Redis is used as an **execution pipeline**.

Detailed architectural decisions are documented separately.

---

## Documentation

All detailed documentation from the original README has been preserved and moved into dedicated files:

- [Architecture & Execution Flow](docs/ARCHITECTURE.md)
- [Services Overview](docs/SERVICES.md)
- [API Documentation](docs/API.md)
- [Running Locally & Environment Variables](docs/RUNNING.md)
- [Shared Libraries](docs/LIBS.md)
- [Testing](TESTING.md)

---

## License

This project is developed for educational and portfolio purposes.
Licensed under the Apache License 2.0.
