package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ddlambda "github.com/DataDog/datadog-lambda-go"
	"github.com/aws/aws-lambda-go/lambda"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type LambdaHeader struct {
	TraceInfo tracer.TextMapCarrier `json:"trace_info"`
}

type PayloadData struct {
	Data   interface{}  `json:"data"`
	Header LambdaHeader `json:"header"`
}
type HelloPayload struct {
	StrData string `json:"str_data"`
	IntData int    `json:"int_data"`
}

func main() {
	cfg := &ddlambda.Config{
		// we can manually extract tracing data from tracing injection
		//TraceContextExtractor: traceContextExtractor,
		DDTraceEnabled: true,
	}

	// Wrap your lambda handler
	lambda.Start(ddlambda.WrapFunction(HandleRequest, cfg))
}

func traceContextExtractor(ctx context.Context, ev json.RawMessage) map[string]string {
	fmt.Printf("eventData: %+v \n\n", string(ev))
	headers := LambdaHeader{}
	err := json.Unmarshal(ev, &headers)
	if err != nil {
		return map[string]string{}
	}
	fmt.Printf("\n\n====\nheader: %+v \n\n", headers.TraceInfo)
	return headers.TraceInfo
}

func HandleRequest(ctx context.Context, payloadData PayloadData) (*string, error) {
	//#region logging and using extractor
	//fmt.Printf("old context_content: %+v \n\n", ctx)
	//if ev == nil {
	//	return nil, fmt.Errorf("received nil event")
	//}
	//fmt.Printf("\n============\ndata: %+v", string(*ev))
	//newCtx := ctx

	//// because current context is not include traceID, spanID, so we can't se
	//span, _ := tracer.StartSpanFromContext(ctx, "lambda HandleRequest 3333")
	//defer span.Finish()
	//span.SetTag("exec lambda", "hello")
	//#endregion logging and using extractor
	fmt.Printf("handle request payload data: %+v", payloadData)

	newCtx := handleTracingManual(ctx, payloadData.Header)
	fmt.Printf("handleTracingManual newContext outside: %+v \n===========\n", newCtx)
	spanCtx1 := childLambda(newCtx, "1")
	childLambda(newCtx, "2")
	childLambda(spanCtx1, "3")
	fmt.Printf("\n===========\n start sleeeepy")
	time.Sleep(2 * time.Second)
	fmt.Printf("\n===========\n fnish sleeeepy")

	data := HelloPayload{}

	payloadDataJson, err := json.Marshal(payloadData.Data)
	if err != nil {
		fmt.Printf("can not json.Marshal(payloadData.Data) :%v", err)
		return nil, err
	}
	err = json.Unmarshal(payloadDataJson, &data)
	if err != nil {
		fmt.Printf("can not json.Unmarshal(payloadDataJson, &data): %v", err)
		return nil, err
	}

	return &data.StrData, nil
}

func childLambda(ctx context.Context, name string) context.Context {
	span, newCtx := tracer.StartSpanFromContext(ctx, fmt.Sprintf("lambda childLambda %v", name))
	defer span.Finish()
	time.Sleep(1 * time.Second)
	span.SetTag("exec childLambda", fmt.Sprintf("lambda childLambda %v", name))
	return newCtx
}

func handleTracingManual(ctx context.Context, event interface{}) (traceCtx context.Context) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("can not marshal event: %+v", err)
		return ctx
	}
	headers := LambdaHeader{}
	err = json.Unmarshal(eventJSON, &headers)
	if err != nil {
		fmt.Printf("can not Unmarshal event data to textMapCarrier: %+v", err)
		return ctx
	}
	fmt.Printf("\n\n====\nHandleRequest header: %+v \n\n", headers.TraceInfo)

	spanCtx, err := tracer.Extract(headers.TraceInfo)
	if err != nil {
		fmt.Errorf("can not tracer.Extract(textMapCarrier)  textMapCarrier: %+v; err: %+v \n\n", headers.TraceInfo, err)
		return ctx
	}
	newSpan := tracer.StartSpan("Get tracing context", tracer.ChildOf(spanCtx))
	defer newSpan.Finish()
	fmt.Printf("handleTracingManual newContext: %+v \n===========\n", ctx)
	ctx = tracer.ContextWithSpan(ctx, newSpan)
	return ctx
}
