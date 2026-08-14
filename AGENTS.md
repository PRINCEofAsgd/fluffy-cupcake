# 工作要求

每次工作都先参考本文档，并遵循以下规则进行工作。

## 理解项目

默认按以下顺序，按需结合项目内容理解项目，若有指定内容可以优先理解指定文档

- 说明文档 [README.md](README.md)
- 结构文档 [docs/structures/repository-structure.md](docs/structures/repository-structure.md)
- 调用链文档 [docs/structures/call-chains.md](docs/structures/call-chains.md)
- 数据库开发规范文档 [docs/foundation/db-dev-rules.md](docs/foundation/db-dev-rules.md)
- 数据库结构文档 [docs/structures/database-structure.md](docs/structures/database-structure.md)
- 脚本文档 [scripts/README.md](scripts/README.md)
- 开发文档 [docs/user/development.md](docs/user/development.md)
- 工作上下文文档 [docs/record/project-context.md](docs/record/project-context.md)
- 操作手册文档 [docs/user/user.md](docs/user/user.md)
- 项目源码 
- 其他相关项目规则文档索引(暂无)
- 版本说明文档
    - [docs/user/release-notes.md](docs/user/release-notes.md)
- 项目工作记录目录 [docs/record/](docs/record/)

其他工作相关文档
- [.dockerignore](.dockerignore)
- [.gitignore](.gitignore)



### 文档职责

- 说明文档为常规项目介绍
- 上下文文档记录当前阶段工作内容记录，项目历史工作记录目录 `docs/record/context-archives/` 按工作阶段归档整个阶段的上下文文档
- 结构文档记录文档索引、项目目录结构(代码、文档、配置等都要记录)、重要子目录结构内部文件具体功能
- 调用链文档记录完整运行项目产物程序各个功能时项目中各个代码模块的调用链和功能作用
- 数据库结构文档记录数据库表结构，包括表名、字段名、字段类型、字段约束等
- 版本说明文档分不同平台独立记录产物的当前版本相关信息，不记录历史版本描述
- 开发文档记录面向开发者的依赖安装/卸载、编译、运行、打包方法指南
- 操作手册文档记录面向使用者的使用方法指南



## 工作方式

按以下工作情形选择工作边界

- 若要求分析或计划 bug 修复、迭代技术选型，则非特定要求不对项目文件进行修改
- 若要求修改迭代：
    - 编写代码过程中对每个文件的功能和每个函数的功能和部分重要代码处写中文注释
    - 如有需要，编写完对说明文档、结构文档、调用链文档、数据库结构文档、开发文档、版本说明文档、操作手册文档、.dockerignore、.gitignore进行更新
    - 将工作内容记录在上下文文档的新一个或下一个 Step 中，新 Step 记录在旧 Step 的上方


### 上下文文档处理

按以下工作情形选择处理方式

- 若一次需要记录 Step 的工作内容是迭代一个新方向或解决一个新问题，则在一个新增的 Step 中记录
- 若一次需要记录 Step 的工作内容是延续解决本窗口或上一个 Step 中未解决完的问题、处理不当的修改、衍生的子问题，则在原有同一个的 Step 中记录
- 只有在明确要求时，将上下文文档中一个阶段的工作内容记录归档到项目历史工作记录目录下


### 版号更新

若要求工作是修改迭代，则在工作完后按以下要求进行版号更新

- 将更新内容以最简短的话语描述在版本说明文档的 "本次更新" 板块中，只记录用户能明显体验到的改动
- 根据版号规则进行版号更新：主版本.次版本.修订版本_日期
    - 主版本：没有明确要求一般不增加，第一次为0
    - 次版本：进行大功能迭代时明确要求了归档阶段上下文文档时增加
    - 修订版本：每次修复 bug 时增加
    - 日期：每次工作完后修改
    - 例：V1.3.8_20260617
- 如有需要，对具体平台产物的整体功能描述进行修改
- 根据结构文档最后中记录的所有版号相关文档、打包脚本、代码等内容，进行所有相关版号记录的更新