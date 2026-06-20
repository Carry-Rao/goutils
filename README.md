我们提供[中文](https://github.com/Carry-Rao/goutils/blob/master/docs/README_zh.md)

# Goutils — Simple to Use, Seamless to Switch

This project is designed to reduce development adaptation costs and unify coding specifications. Its core concepts are as follows:

- Encapsulate complex underlying implementations to lower the threshold for integration
- Support seamless switching between multiple platforms and protocols for diverse business scenarios
- Provide standardized and unified calling interfaces to unify coding styles
- Encapsulate frequently used and repetitive functions to eliminate redundant code

+ **However**, my programming proficiency is insufficient, and almost none of the modules have undergone testing. Deployment in production environments is discouraged.

## Database Module
This module features **low intrusion and seamless switching**. Developers can flexibly adapt to various mainstream databases without modifying business code. It natively supports the following databases:

- PostgreSQL
- MySQL
- SQLite

In addition, cache components can be used as data storage carriers to expand lightweight data read-write capabilities for cache-based business scenarios:

- Redis Cache
- In-memory Cache
- Bloom Filter

Furthermore, the project has a built-in **Mixture Composite Database** that supports multi-layer data link combination and customized error handling strategies, with core capabilities as follows:

- Customizable multi-layer data links (e.g., Filter → Cache → Database) to adapt to multi-level reading, writing, filtering and degradation scenarios
- Configurable diverse error handling strategies, including automatic degradation and error interception, to improve service stability
- Consistent unified calling interfaces throughout, ensuring link and strategy changes do not affect upper-level business code

[Complete API Documentation](https://github.com/Carry-Rao/goutils/blob/master/docs/database_en.md)

## Log Module

This module provides a **unified logging capability**, supporting multi-level logging, buffered writes, and colored output. Its core features include:

- **Multi-level Logging**: Supports DEBUG, INFO, WARN, ERROR, PANIC five log levels
- **Buffered Write**: Buffer mechanism to reduce frequent I/O and improve performance
- **Color Output**: Supports terminal color output for easy log level identification
- **Auto Flush**: PANIC and FATAL level logs automatically flush the buffer and terminate the program
- **Level Filtering**: Control log output granularity by setting the log level

[Complete API Documentation](https://github.com/Carry-Rao/goutils/blob/master/docs/log_en.md)
