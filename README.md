# 基于 RAG 的知识库应用

1. **网页爬取与文本处理**：抓取网页内容，并进行分块处理。
2. **Embedding 向量化**：使用 `nomic-embed-text:latest` 进行文本向量化，将文本转换为向量表示。
3. **向量数据库存储**：利用 PgVector 存储文本的向量表示。
4. **LLM 生成回答**：用户输入问题，基于检索出的相关文档，由 `deepseek-r1:1.5b` 生成最终的回答。

启动应用时，需要提供网页 URL，例如：

```shell
$ go run main.go https://kubernetes.io/zh-cn/blog/2024/12/17/kube-apiserver-api-streaming/
成功将 https://kubernetes.io/zh-cn/blog/2024/12/17/kube-apiserver-api-streaming/ 拆分为 25 个块
正在添加文档到向量存储: 100.0% (25/25)
成功加载 25 个文档片段到向量存储

请输入您的问题>>> 启用 API 流式传输需要什么版本

找到 3 条相关文档：

1. [来源：https://kubernetes.io/zh-cn/blog/2024/12/17/kube-apiserver-api-streaming/, 分块：20, 相似度：0.70]
   摘要：内存是被突然且快速分配的，仅仅几个请求就可能迅速耗尽可用内存，导致�...

2. [来源：https://kubernetes.io/zh-cn/blog/2024/12/17/kube-apiserver-api-streaming/, 分块：21, 相似度：0.67]
   摘要：API 服务器受限于 etcd 中对象的最大允许体量加上少量额外分配的内存。 与��...

3. [来源：https://kubernetes.io/zh-cn/blog/2024/12/17/kube-apiserver-api-streaming/, 分块：0, 相似度：0.66]
   摘要：文档Kubernetes 博客培训合作伙伴社区案例分析版本列表发布信息 v1.32 v1.31 v1.3...

生成回答中...

<think>
嗯，用户的问题是关于启用API流式传输需要什么Kubernetes版本。我得仔细看看提供的文档内容来回答。

首先，文档里提到在Kubernetes 1.32中，有“watch list 特性进阶至 Beta”，并且提供了通过Switch List请求成为watch request的信息。这说明现在支持流式处理list请求，用户可以将这些请求切换为watch形式。

然后，在第二部分详细讨论了如何使用watch请求，并提到这种新的方法可以保持恒定的内存开销，避免资源耗尽的问题。这进一步确认了流式处理的重要性，特别是在 高可用集群中的应用。

最后，文档还提到了版本信息v1.32及以上，说明启用这一功能需要至少Kubernetes 1.32版本才能运行。

总结一下，用户可以通过调整List请求为Watch形式，从而实现流式处理，而启用这一点需要Kubernetes 1.32或更高版本。
</think>

在 Kubernetes 中，如果想启用 API 流式传输，并且希望使用 Watch List 特性进阶至 Beta 版本（即 `watch list 特性进阶至 Beta`），您可以通过以下步骤实现：

1. **调整请求类型**：将列表请求（List Request）切换为Watch Request。通过在显式启用 Watch List 特性门控后，将列表请求切换为某种特殊类型的 `watch request`。

2. **Kubernetes 版本要求**：为了实现流式处理，并且支持进阶特性，您需要 Kubernetes 的 1.32 版本或更高版本。文档中提到在 Kubernetes 1.32 中有“watch list 特性进阶至 Beta”，并且提供了通过 Switch List 请求进行 watch 调整的信息。

总结：启用 API 流式传输需要使用 Kubernetes 1.32 版本或更高，并且支持进阶特性。

请输入您的问题>>>
```
