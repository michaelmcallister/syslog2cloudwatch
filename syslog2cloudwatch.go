package main

import (
    "fmt"
    "strings"
    "time"
    "os"
    "gopkg.in/mcuadros/go-syslog.v2"
    "github.com/spf13/viper"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func main() {
    c, err := initConfig()
    if err != nil {
        fmt.Println("Init error:", err)
        return
    }

	channel, server, err := initServer()
	if err != nil {
        fmt.Println("Server error:", err)
        return
    }


    go func(channel syslog.LogPartsChannel) {
        for logParts := range channel {
            syslogMessage := make([]string, 0, len(logParts))

            for key, value := range logParts {
                message := fmt.Sprintf("%s:%v", key, value)
                syslogMessage = append(syslogMessage, message)
            }
                fmt.Println(syslogMessage)
                c.putLog(strings.Join(syslogMessage,", "))
        }
    }(channel)

    server.Wait()
}

type CW struct {
	svc        *cloudwatchlogs.CloudWatchLogs
	groupName, streamName  string
	token      *string
}

func initConfig() (*CW, error) {
    os.Setenv("AWS_SDK_LOAD_CONFIG", "1")

    userConfig := viper.New()
    userConfig.SetConfigName("app")
    userConfig.AddConfigPath(".")
    userConfig.AddConfigPath("config")
    err := userConfig.ReadInConfig()
    if err != nil {
        return nil, err
    }

    sess, err := session.NewSession()
    if err!= nil {
        return nil, err
    }
    c := &CW{
            svc:           cloudwatchlogs.New(sess),
            groupName:     userConfig.GetString("loggroup"),
            streamName:    userConfig.GetString("logstream"),
        }

    return c, c.getToken()
}

func initServer() (syslog.LogPartsChannel, *syslog.Server, error){
    channel := make(syslog.LogPartsChannel)
    handler := syslog.NewChannelHandler(channel)
    server := syslog.NewServer()
    server.SetFormat(syslog.RFC3164)
    server.SetHandler(handler)

    err := server.ListenUDP("0.0.0.0:514")
	if err != nil {
        return nil, nil, err
    }

    err = server.ListenTCP("0.0.0.0:514")
    if err != nil {
        return nil, nil, err
    }

    return channel, server, server.Boot()
}

func (c *CW) getToken() (error) {
    resp, err := c.svc.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
        LogGroupName:        aws.String(c.groupName),
        LogStreamNamePrefix: aws.String(c.streamName),
    })

    if err != nil {
        return err
    }

    if len(resp.LogStreams) > 0 {
        c.token = resp.LogStreams[0].UploadSequenceToken
    }
    return nil
}

func (c *CW) putLog(event string) (int, error) {
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
        } else {
            fmt.Println(err)
        }
        return len(event), err
}
