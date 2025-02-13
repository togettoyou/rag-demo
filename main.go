package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
)

const (
	// DefaultOllamaServer 默认的Ollama服务器地址
	DefaultOllamaServer = "http://localhost:11434"
	// DefaultEmbeddingModel 用于生成文本向量的默认模型
	DefaultEmbeddingModel = "nomic-embed-text:latest"
	// DefaultLLMModel 用于生成回答的默认大语言模型
	DefaultLLMModel = "deepseek-r1:1.5b"
	// DefaultPGVectorURL PostgreSQL向量数据库的连接URL
	DefaultPGVectorURL = "postgres://pgvector:pgvector@localhost:5432/llm-test?sslmode=disable"
)

func must(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	// 解析命令行参数中的URL并加载网页内容
	allDocs, err := parseAndloadDocumentsFromURLs()
	must(err)

	// 初始化文本向量化模型
	embedder, err := initEmbedder()
	must(err)

	// 初始化向量数据库
	store, err := initVectorStore(embedder)
	must(err)

	// 将文档添加到向量数据库
	addDocumentsToStore(store, allDocs)

	// 初始化大语言模型
	llm, err := initLLM()
	must(err)

	// 启动交互式问答
	startInteractiveQA(store, llm)
}

func parseAndloadDocumentsFromURLs() ([]schema.Document, error) {
	// 检查命令行参数，确保至少提供了一个URL
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("请指定至少一个网页URL")
	}
	urls := os.Args[1:]

	var allDocs []schema.Document
	// 遍历处理每个URL
	for _, url := range urls {
		// 加载并分割网页内容
		docs, err := loadAndSplitWebContent(url)
		if err != nil {
			fmt.Printf("加载网页 %s 失败: %v\n", url, err)
			// 继续处理下一个URL
			continue
		}
		// 将当前URL的文档添加到总文档集合中
		allDocs = append(allDocs, docs...)
		fmt.Printf("成功将 %s 拆分为 %d 个块\n", url, len(docs))
	}

	// 确保至少成功加载了一个网页
	if len(allDocs) == 0 {
		return nil, fmt.Errorf("没有成功加载任何网页")
	}
	return allDocs, nil
}

func loadAndSplitWebContent(url string) ([]schema.Document, error) {
	// 发送HTTP GET请求获取网页内容
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 使用goquery解析HTML文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var content strings.Builder

	// 移除script和style标签，避免抓取无关内容
	doc.Find("script,style").Remove()
	// 提取body中的所有文本内容
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			content.WriteString(text)
			content.WriteString("\n")
		}
	})

	// 将文本分割成多个块，设置块大小为512字符，无重叠
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(512),
		textsplitter.WithChunkOverlap(0),
	)
	chunks, err := splitter.SplitText(content.String())
	if err != nil {
		return nil, err
	}

	// 为每个文本块创建Document对象，包含元数据
	documents := make([]schema.Document, 0)
	for i, chunk := range chunks {
		documents = append(documents, schema.Document{
			PageContent: chunk,
			Metadata: map[string]any{
				"source": url,                  // 记录文本来源URL
				"chunk":  fmt.Sprintf("%d", i), // 记录块的序号
			},
		})
	}
	return documents, nil
}

func initEmbedder() (embeddings.Embedder, error) {
	embedModel, err := ollama.New(
		ollama.WithServerURL(DefaultOllamaServer),
		ollama.WithModel(DefaultEmbeddingModel),
	)
	if err != nil {
		return nil, fmt.Errorf("创建embedding模型失败: %v", err)
	}

	embedder, err := embeddings.NewEmbedder(embedModel)
	if err != nil {
		return nil, fmt.Errorf("初始化embedding模型失败: %v", err)
	}
	return embedder, nil
}

func initVectorStore(embedder embeddings.Embedder) (vectorstores.VectorStore, error) {
	store, err := pgvector.New(
		context.Background(),
		pgvector.WithConnectionURL(DefaultPGVectorURL),
		pgvector.WithEmbedder(embedder), // 绑定向量模型
		pgvector.WithCollectionName(uuid.NewString()),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化向量存储失败: %v", err)
	}
	return &store, nil
}

func addDocumentsToStore(store vectorstores.VectorStore, allDocs []schema.Document) {
	// 设置批处理大小，避免一次处理太多文档
	batchSize := 10
	totalDocs := len(allDocs)
	processedDocs := 0

	// 分批处理所有文档
	for i := 0; i < totalDocs; i += batchSize {
		end := i + batchSize
		if end > totalDocs {
			end = totalDocs
		}

		batch := allDocs[i:end]
		// 将文档添加到向量存储
		_, err := store.AddDocuments(context.Background(), batch)
		if err != nil {
			fmt.Printf("\n添加文档到向量存储失败: %v\n", err)
			continue
		}

		processedDocs += len(batch)
		progress := float64(processedDocs) / float64(totalDocs) * 100
		fmt.Printf("\r正在添加文档到向量存储: %.1f%% (%d/%d)", progress, processedDocs, totalDocs)
	}
	fmt.Printf("\n成功加载 %d 个文档片段到向量存储\n", totalDocs)
}

func initLLM() (llms.Model, error) {
	llm, err := ollama.New(
		ollama.WithServerURL(DefaultOllamaServer),
		ollama.WithModel(DefaultLLMModel),
	)
	if err != nil {
		return nil, fmt.Errorf("初始化LLM失败: %v", err)
	}
	return llm, nil
}

func startInteractiveQA(store vectorstores.VectorStore, llm llms.Model) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n请输入您的问题>>> ")
		question, _ := reader.ReadString('\n')
		question = strings.TrimSpace(question)

		handleQuestion(store, llm, question)
	}
}

func handleQuestion(store vectorstores.VectorStore, llm llms.Model, question string) {
	// 在向量数据库中搜索相关文档
	// 参数：最多返回5个结果，相似度阈值0.7
	results, err := store.SimilaritySearch(
		context.Background(),
		question,
		5,
		vectorstores.WithScoreThreshold(0.7),
	)
	if err != nil {
		fmt.Printf("搜索相关文档失败: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("\n未找到相关的参考信息，请换个问题试试。")
		return
	}

	// 显示检索到的文档
	displaySearchResults(results)
	// 将相关文档作为上下文提供给大语言模型并生成问题的回答
	generateAnswer(llm, question, results)
}

func displaySearchResults(results []schema.Document) {
	fmt.Printf("\n找到 %d 条相关文档：\n", len(results))
	for i, doc := range results {
		score := 1 - doc.Score
		fmt.Printf("\n%d. [来源：%v, 分块：%v, 相似度：%.2f]\n", i+1,
			doc.Metadata["source"],
			doc.Metadata["chunk"],
			score,
		)

		content := doc.PageContent
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("   摘要：%s\n", strings.ReplaceAll(content, "\n", " "))
	}
	fmt.Println()
}

func generateAnswer(llm llms.Model, question string, results []schema.Document) {
	var references strings.Builder
	for i, doc := range results {
		score := 1 - doc.Score
		references.WriteString(fmt.Sprintf("%d. [相似度：%f] %s\n", i+1, score, doc.PageContent))
	}

	messages := []llms.MessageContent{
		{
			// 系统提示，设置助手角色和行为规则
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextContent{
					Text: fmt.Sprintf(
						"你是一个专业的知识库问答助手。以下是基于向量相似度检索到的相关文档：\n\n%s\n"+
							"请基于以上参考信息回答用户问题。回答时请注意：\n"+
							"1. 优先使用相关度更高的参考信息\n"+
							"2. 如果参考信息不足以完整回答问题，请明确指出",
						references.String(),
					),
				},
			},
		},
		{
			// 用户问题
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextContent{
					Text: question,
				},
			},
		},
	}

	fmt.Printf("生成回答中...\n\n")

	_, err := llm.GenerateContent(
		context.Background(),
		messages,
		llms.WithTemperature(0.8), // 设置温度为0.8，增加回答的多样性
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			fmt.Print(string(chunk))
			return nil
		}),
	)
	if err != nil {
		fmt.Printf("生成回答失败: %v\n", err)
		return
	}

	fmt.Println()
}
