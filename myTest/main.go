package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	//f, err := os.OpenFile("./mylog/application.logger", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	//if err != nil {
	//	panic(err)
	//}
	// Log as JSON instead of the default ASCII formatter.
	logger.SetFormatter(&logger.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logger.SetOutput(os.Stdout)
	//logger.SetOutput(f)

	// Only logger the warning severity or above.
	logger.SetLevel(logger.InfoLevel)
}

func main() {
	logger.Info("start application")
	tracer.Start(
		tracer.WithEnv("dev-env"),
		tracer.WithService("lambda-tracing-client-test"),
		tracer.WithServiceVersion("12"),
		tracer.WithAgentAddr("localhost:8126"),
	)
	logger.Info("Hellow rold: ", time.Now())
	// When the tracer is stopped, it will flush everything it has to the Datadog Agent before quitting.
	// Make sure this line stays in your main function.
	defer tracer.Stop()

	ctx := context.Background()
	newspan, ctx := tracer.StartSpanFromContext(ctx, "main func")
	defer newspan.Finish()
	newspan.SetTag("main_key", "main_value")
	//
	abc(ctx)
	time.Sleep(1 * time.Second)
	def(ctx)
	logger.Info("end main func")
	InvokeLambda(ctx)

}

type PayloadData struct {
	Hello string `json:"hello"`
	LambdaHeader
}

type LambdaHeader struct {
	TraceInfo tracer.TextMapCarrier `json:"trace_info"`
}

func InvokeLambda(ctx context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, "invoke lambda func")
	defer span.Finish()
	span.SetTag("my_spanID", span.Context().SpanID())
	span.SetTag("my_traceID", span.Context().TraceID())
	fmt.Printf("spanID: %v\n", span.Context().SpanID())
	fmt.Printf("traceID: %v\n", span.Context().TraceID())
	// --- invoke lambda
	injectedSpanCtx, ok := span.Context().(ddtrace.SpanContext)
	if !ok {
		return
	}
	tracerCarrier := make(tracer.TextMapCarrier)
	err := tracer.Inject(injectedSpanCtx, tracerCarrier)
	if err != nil {
		return
	}

	tracerCarrierByte, err := json.Marshal(tracerCarrier)
	if err != nil {
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile("aa_franky"),
	)
	if err != nil {
		fmt.Println("Error loading AWS SDK config:", err)
		return
	}

	// Create Lambda client
	client := lambda.NewFromConfig(cfg)
	tracerCarrierByteStr := string(tracerCarrierByte)
	fmt.Println("traceInfo: ", tracerCarrierByteStr)

	// Lambda function name and payload
	functionName := "franky-lambda-tracing-dev-helloWorld"
	payload := PayloadData{
		Hello:        "hello hihi 1",
		LambdaHeader: LambdaHeader{TraceInfo: tracerCarrier},
	}

	payloadJSON, _ := json.Marshal(payload)

	//clientContextBase64 := base64.StdEncoding.EncodeToString(tracerCarrierByte)

	// Invoke Lambda function
	result, err := client.Invoke(ctx, &lambda.InvokeInput{
		//ClientContext: aws.String(clientContextBase64),
		FunctionName: &functionName,
		Payload:      payloadJSON,
	})

	if err != nil {
		fmt.Println("Error invoking Lambda function:", err)
		return
	}

	// Print the result
	fmt.Println("Lambda function response:", string(result.Payload))
}

func abc(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "abc func")
	span.SetTag("abc func key", "abc func value")
	//time.Sleep(time.Duration(2) * time.Second)
	defer span.Finish()
}

func def(ctx context.Context) {
	span, _ := tracer.StartSpanFromContext(ctx, "def func")
	span.SetTag("def func key", "def func value")
	//time.Sleep(time.Duration(1) * time.Second)
	defer span.Finish()
}
