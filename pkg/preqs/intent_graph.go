package processreqs

import (
	"fmt"
	"strings"

	pb "github.com/digital-dream-labs/api/go/chipperpb"
	"cavalier/pkg/vars"
	"cavalier/pkg/vtt"
	sr "cavalier/pkg/speechrequest"
	ttr "cavalier/pkg/ttr"
)

func (s *Server) ProcessIntentGraph(req *vtt.IntentGraphRequest) (*vtt.IntentGraphResponse, error) {
	var successMatched bool
	speechReq := sr.ReqToSpeechRequest(req)
	var transcribedText string
	
	if !isSti {
		var err error
		transcribedText, err = sttHandler(speechReq)
		if err != nil {
			ttr.IntentPass(req, "intent_system_noaudio", "voice processing error", map[string]string{"error": err.Error()}, true)
			return nil, nil
		}
		if strings.TrimSpace(transcribedText) == "" {
			ttr.IntentPass(req, "intent_system_noaudio", "", map[string]string{}, false)
			return nil, nil
		}
		successMatched = ttr.ProcessTextAll(req, transcribedText, vars.IntentList, speechReq.IsOpus)
	} else {
		intent, slots, err := stiHandler(speechReq)
		if err != nil {
			if err.Error() == "inference not understood" {
				fmt.Println("Bot " + speechReq.Device + " No intent was matched")
				ttr.IntentPass(req, "intent_system_unmatched", "voice processing error", map[string]string{"error": err.Error()}, true)
				return nil, nil
			}
			fmt.Println(err)
			ttr.IntentPass(req, "intent_system_noaudio", "voice processing error", map[string]string{"error": err.Error()}, true)
			return nil, nil
		}
		ttr.ParamCheckerSlotsEnUS(req, intent, slots, speechReq.IsOpus, speechReq.Device)
		return nil, nil
	}
	
	if !successMatched {
		fmt.Println("No intent was matched.")
		
		if vars.APIConfig.Knowledge.Enable && vars.APIConfig.Knowledge.Provider == "houndify" && len([]rune(transcribedText)) >= 8 {
			fmt.Println("Making Houndify request for device " + req.Device + "...")
			
			InitKnowledge()
			
			apiResponse := houndifyKG(speechReq)
			
			if apiResponse != "" && apiResponse != "Houndify is not enabled." {
				response := &pb.IntentGraphResponse{
					Session:      req.Session,
					DeviceId:     req.Device,
					ResponseType: pb.IntentGraphMode_KNOWLEDGE_GRAPH,
					SpokenText:   apiResponse,
					QueryText:    transcribedText,
					IsFinal:      true,
				}
				
				if err := req.Stream.Send(response); err != nil {
					fmt.Println("Error sending IntentGraph response:", err)
					return nil, err
				}
				
				fmt.Println("Bot " + req.Device + " Houndify IntentGraph request served.")
				return nil, nil
			} else {
				fmt.Println("No valid response from Houndify")
			}
		}
		
		ttr.IntentPass(req, "intent_system_unmatched", transcribedText, map[string]string{"": ""}, false)
		return nil, nil
	}
	
	fmt.Println("Bot " + speechReq.Device + " request served.")
	return nil, nil
}