---
name: Java 服务
description: 用 Java 写后端服务时使用。约束 Maven/Gradle 工程结构、分层、异常与配置，产出可以打包运行的 Spring Boot 应用。
---

# Java 服务

## 工程形态

优先用 Spring Boot 3 + Maven。

```text
src/main/java/<group>/<app>/
  Application.java
  controller/        HTTP 入口
  service/           业务逻辑
  repository/        数据访问
  model/             实体与 DTO
src/main/resources/
  application.yml
```

- Java 版本在 `pom.xml` 里写死，不要依赖机器默认版本。
- 配置分环境：`application.yml` + `application-dev.yml`，密钥用环境变量占位。
- 产物是 `target/*.jar`，能直接 `java -jar` 启动。

## 必守规则

1. **Controller 不写业务。** 只做参数绑定、调用 service、返回 DTO。
2. **异常集中处理。** 用 `@RestControllerAdvice` 把业务异常翻译成统一响应，不要每个方法 try-catch 一遍。
3. **注入用构造器。** 不用字段 `@Autowired`。
4. **DTO 与实体分开。** 不要把 Entity 直接返给前端。
5. **日志用 SLF4J。** 禁止 `System.out.println`。

## 常写片段

```java
@RestController
@RequestMapping("/api/orders")
public class OrderController {
    private final OrderService service;

    public OrderController(OrderService service) {
        this.service = service;
    }

    @GetMapping
    public List<OrderDto> list(@RequestParam(defaultValue = "20") int limit) {
        return service.list(limit);
    }
}
```

## 验收

- `mvn -q package` 或 `./gradlew build` 通过。
- `java -jar target/*.jar` 能起来，端口无冲突。
- 主要接口用 `curl` 打一遍，返回结构符合约定。
