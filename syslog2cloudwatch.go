package main

import (
    "fmt"
    "time"
    "gopkg.in/mcuadros/go-syslog.v2"
    "github.com/spf13/viper"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func main() {
    c, err := initConfig()
    if err != nil {
        fmt.Println("Init error: ", err)
    }

    channel, server := initServer()
    go func(channel syslog.LogPartsChannel) {
        for logParts := range channel {
            c.putLog(logParts["message"].(string))
        }
    }(channel)

    server.Wait()
}

type CW struct {
	svc                    *cloudwatchlogs.CloudWatchLogs
	groupName, streamName  string
	token      *string
}

func initConfig() (*CW, error) {
    userConfig := viper.New()
    userConfig.SetConfigName("app")
    userConfig.AddConfigPath(".")
    userConfig.AddConfigPath("config")
    err := userConfig.ReadInConfig()
    if err != nil {
        return nil, err
    }
    c := &CW{
            svc:           cloudwatchlogs.New(session.New()),
            groupName:     userConfig.GetString("loggroup"),
            streamName:    userConfig.GetString("logstream"),
        }
    c.getToken()
    return c, nil
}

func initServer() (syslog.LogPartsChannel, *syslog.Server){
    channel := make(syslog.LogPartsChannel)
    handler := syslog.NewChannelHandler(channel)
    server := syslog.NewServer()
    server.SetFormat(syslog.RFC5424)
    server.SetHandler(handler)
    server.ListenUDP("0.0.0.0:514")
    server.ListenTCP("0.0.0.0:514")

    server.Boot()
    return channel, server
}

func (c *CW) getToken() (int, error) {
    resp, err := c.svc.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
        LogGroupName:        aws.String(c.groupName),
        LogStreamNamePrefix: aws.String(c.streamName),
    })

    if err != nil {
        return 1, err
    }

    if len(resp.LogStreams) > 0 {
        c.token = resp.LogStreams[0].UploadSequenceToken
    }
    return 0, nil
}

func (c *CW) putLog(event string) (int, error) {
        fmt.Println(c.token)
        params := &cloudwatchlogs.PutLogEventsInput{
                LogEvents: []*cloudwatchlogs.InputLogEvent{
                        {
                                Message:   aws.String(event),
                                Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
                        },
                },
                LogGroupName:  aws.String(c.groupName),
                LogStreamName: aws.String(c.streamName),
                SequenceToken: c.token,
        }
        resp, err := c.svc.PutLogEvents(params)
        if err == nil {
            c.token = resp.NextSequenceToken
            fmt.Println(resp)
        } else {
            fmt.Println(err)
        }
        return len(event), err
}
