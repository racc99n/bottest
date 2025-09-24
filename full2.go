package main

import (
	"bytes"
	"container/heap"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	_ "image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	_ "image/jpeg" // import jpeg decoder

	_ "image/png" // Import png decoder

	_ "github.com/go-sql-driver/mysql"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"github.com/tuotoo/qrcode"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	LineChannelAccessToken = "ktbmUgNd3JrYW5mv8uL/RqYEKBCoL7IQkCQFy407454zebPQHPSMrandf+EGhv7HAJXYR3NlrtycFZMt8R0Vu5hG+bYc0ieowvK/tldMYLA1pCBY7U6oM2Veh0myF6ziSzpsQwbTbXCx+ahI23T8TAdB04t89/1O/w1cDnyilFU="
	LineChannelSecret      = "fb1092f426235281c348081af81cd1b0"
	liffProfile            = "https://liff.line.me/2007195340-OVkVxXzG"
	liffSummary            = "https://liff.line.me/2007195340-WeXArK3d"
	liffGame               = "https://liff.line.me/2007195340-lDpQywXZ"
	liffClear              = "https://liff.line.me/2007195340-BrMQ0rbq"
	liffHQ                 = "https://liff.line.me/2007195340-6bJd37pP"
	liffRealTime           = "https://liff.line.me/2007195340-1MpwZGJV"
	// liffSummary            = "https://liff.line.me/2006741358-oG3lxbGE"
	liffDev101    = "https://liff.line.me/2007195340-OVkVxXzG"
	BankAcc       = "1501407093"
	BankAcc2      = "020345844722"
	BankName2     = "ธนาคารออมสิน"
	BankName3     = "ธนาคารกรุงเทพ"
	BankName      = "ธนาคารกสิกรไทย"
	PlayRoom      = "https://line.me/ti/g/efxQ4x-z8L"
	DepositRoom   = "https://line.me/ti/g/2MHVJf5p_9"
	BankURL       = "https://static.thairath.co.th/media/dFQROr7oWzulq5Fa5yCh2MqXjvdMvGdUrJriRli6kTT86e0guCzK0nQKCCdWFSmlhL1.jpg"
	BankURL3      = "https://www.basengineer.com/bas2019/wp-content/uploads/2018/04/BangkokBankThai-750x450.jpg"
	BankURL2      = "https://img.kapook.com/u/2021/surauch/picture2/1_16.jpg"
	UserAccName   = "น.ส. ชมพูนุช ปัดทอง"
	UserAccName2  = "วิทยา นะวะคำ"
	groupPlay     = "Cba4028d48143cbf51b7992098c60239a"
	groupC        = "C848440a6316235fdcb32a19f11f9eb91"
	GroupT        = "C06bb64a1483820fffadced86b6ef21f91"
	adminSLIP     = "U9cae1477dad810d306c1505d2634fa86"
	showE         = false
	showO         = true
	liffALL       = "https://duck98.com/ConfigBotfull2/nav.php"
	liffAll2      = "https://duck98.com/ConfigBotfull2/nav.php"
	playON        = true
	translateNew  = true
	translateNew2 = true
	soi           = 0
	showOP        = true
	showFullO     = false
	houseName     = "มวยพักยก วาดี้"
	showReply     = false
	showHigher    = true
)

var sequenceCounter int64 = 0 // global หรือ package-level variable

type MessagePayload struct {
	Type        string               `json:"type"`
	Text        string               `json:"text"`
	QuoteToken  string               `json:"quoteToken,omitempty"`
	FlexMessage *linebot.FlexMessage `json:"flexMessage,omitempty"` // Added FlexMessage field
}
type BotHandler struct {
	Bot   *linebot.Client
	DB    *sql.DB
	Group string
	User  string
	Token string
}

// Global instance of BotHandler

type FlexMessagePayload struct {
	Type       string                 `json:"type"`
	AltText    string                 `json:"altText"`
	Contents   map[string]interface{} `json:"contents"`
	QuoteToken string                 `json:"quoteToken,omitempty"`
}

type Request struct {
	ReplyToken   string
	RawMessage   string
	ReplyMessage string
	QuoteToken   string
	Timestamp    time.Time
	UID          string
	GroupID      string
	Matched      bool
	Sequence     int64 // <- เพิ่ม field นี้สำหรับ tie-breaker
}

type PriorityQueue []*Request

func (pq PriorityQueue) Len() int { return len(pq) }

//	func (pq PriorityQueue) Less(i, j int) bool {
//		if pq[i].Timestamp.Equal(pq[j].Timestamp) {
//			log.Println("EQUAL TIMESTAMPS")
//			return pq[i].Sequence < pq[j].Sequence // tie-breaker
//		}
//		return pq[i].Timestamp.Before(pq[j].Timestamp)
//	}
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Timestamp.Before(pq[j].Timestamp)
}

type LineUserProfile struct {
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl"`
	StatusMessage string `json:"statusMessage"`
}

func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Request))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

var eventQueue = &PriorityQueue{}
var isProcessing = false

var queueMutex = &sync.Mutex{}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	//bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
	var body struct {
		Events []struct {
			ReplyToken string `json:"replyToken"`
			Message    struct {
				Text       string `json:"text"`
				QuoteToken string `json:"quoteToken"`
				Type       string `json:"type"`
				ID         string `json:"id"`
				PackageID  string `json:"packageId,omitempty"` // Optional, only for sticker messages
				StickerID  string `json:"stickerId,omitempty"` // Optional, only for sticker messages
			} `json:"message"`
			Timestamp int64 `json:"timestamp"` // Assuming the timestamp is included in the payload
			Source    struct {
				UserID  string `json:"userId"`  // Assuming this is the user ID field
				GroupID string `json:"groupId"` // Assuming this is the user ID field
				Type    string `json:"type"`    // Assuming this is the type field
			} `json:"source"` // Added Source field to capture sender's information
		} `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	for _, event := range body.Events {
		userID := event.Source.UserID
		groupID := event.Source.GroupID
		sourceType := event.Source.Type
		log.Printf("IMAGE from user: %s, source type: %s, ID Group: %s", userID, sourceType, groupID)
		if event.Message.Type == "image" {
			// Get the image content using the message ID
			messageID := event.Message.ID
			displayName, pictureURL, _, _ := getUserProfile(groupID, userID)
			InsertUser(userID, displayName, pictureURL)
			if groupID == groupPlay || userID == adminSLIP {
				return
			}
			// Define relative path to save the image
			// Generate a timestamp12
			timestamp := time.Now().Format("20060102_150405") // Format: YYYYMMDD_HHMMSS

			// Define relative path with timestamp
			relativePath := fmt.Sprintf("./images/%s_%s.jpg", timestamp, messageID)

			// Initialize an HTTP client
			client := &http.Client{}
			if userID == "U79646e6057021cb18e11fe05bc868e58" || userID == "U2656f5d36059dc51cad5a0c2bfe00deb" {
				return
			}

			// Fetch the image content
			req, err := http.NewRequest("GET", fmt.Sprintf("https://api-data.line.me/v2/bot/message/%s/content", messageID), nil)
			if err != nil {
				log.Printf("Error creating request: %v", err)
				return
			}

			// Add authorization headers (replace "YourChannelAccessToken" with the actual token)
			req.Header.Add("Authorization", "Bearer "+LineChannelAccessToken)

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error fetching image: %v", err)
				return
			}
			defer resp.Body.Close()

			// Check if the response is valid
			if resp.StatusCode != http.StatusOK {
				log.Printf("Failed to fetch image: %s", resp.Status)
				return
			}

			// Save the image to the relative path
			outFile, err := os.Create(relativePath)
			if err != nil {
				log.Printf("Error creating file: %v", err)
				return
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, resp.Body)
			if err != nil {
				log.Printf("Error saving image: %v", err)
				return
			}

			log.Printf("Image saved to: %s", relativePath)
			cmd := exec.Command("/home/james/Documents/FULLOPTION2/.venv/bin/python", "qr.py", userID, relativePath)

			// Capture the output of the Python script
			output, _ := cmd.CombinedOutput()
			// if err != nil {
			// 	log.Println("Error executing Python script: %v", err)
			// }

			// Output from the Python script
			result := string(output)
			fmt.Println("Result from Python script:", result)
			displayName, pictureURL, statusMessage, err := getUserProfile(groupID, userID)

			fmt.Println("Display Name:", displayName)
			// fmt.Println("Picture URL:", pictureURL)
			fmt.Println("Status Message:", statusMessage)
			// credit, credit2, num, err := GetUserData(userID)
			ctx := context.Background()
			// You can process the output further based on your requirements
			if strings.Contains(result, "DUP") {
				// Handle the case where the response is "ABC"
				fmt.Println("DUP")
				actionP := linebot.NewURIAction("ไปห้องเล่น", PlayRoom)
				actionD := linebot.NewURIAction("ไปห้องฝาก", DepositRoom)
				flexMessage := linebot.NewFlexMessage(
					displayName+" เติมเงินไม่สำเร็จ",
					&linebot.BubbleContainer{
						Type: linebot.FlexContainerTypeBubble,
						Size: "deca",
						Header: &linebot.BoxComponent{
							Type:       linebot.FlexComponentTypeBox,
							Layout:     "vertical",
							Height:     "15px",
							Position:   "relative",
							AlignItems: "center",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:      linebot.FlexComponentTypeText,
									Text:      houseName,
									Size:      "xxs",
									Gravity:   "top",
									Align:     "start",
									Position:  "absolute",
									OffsetTop: "xs",
								},
							},
						},
						Body: &linebot.BoxComponent{
							Type:        linebot.FlexComponentTypeBox,
							Layout:      "vertical",
							PaddingAll:  "0px",
							BorderWidth: "normal",
							BorderColor: "#ffc60a",
							Contents: []linebot.FlexComponent{
								&linebot.BoxComponent{
									Type:     "box",
									Layout:   "horizontal",
									Contents: []linebot.FlexComponent{},
								},
								&linebot.BoxComponent{
									Type:       "box",
									Layout:     "horizontal",
									Spacing:    "xs",
									PaddingAll: "20px",
									Contents: []linebot.FlexComponent{
										&linebot.BoxComponent{
											Type:           "box",
											Layout:         "vertical",
											Width:          "55px",
											JustifyContent: "space-between",
											Contents: []linebot.FlexComponent{
												&linebot.BoxComponent{
													Type:         "box",
													Layout:       "vertical",
													CornerRadius: "100px",
													Width:        "48px",
													Height:       "48px",
													BorderWidth:  "medium",
													BorderColor:  "#ff0000",
													Contents: []linebot.FlexComponent{
														&linebot.ImageComponent{
															Type:       "image",
															URL:        "https://png.pngtree.com/png-clipart/20240731/original/pngtree-red-cross-mark-symbol-icon-png-image_15670730.png",
															AspectMode: "cover",
															Size:       "full",
														},
													},
												},
											},
										},
										&linebot.BoxComponent{
											Type:   "box",
											Layout: "vertical",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type: "text",
													Size: "sm",
													Wrap: true,
													Contents: []*linebot.SpanComponent{
														{
															Type:   "span",
															Text:   "สลิปอาจถูกใช้ไปแล้ว" + "\n" + displayName + "\n",
															Weight: "bold",
															Color:  "#ffffff",
															Size:   "xxs",
														},
														{
															Type:   "span",
															Text:   "ธนาคารกรุงเทพ\nต้องรอ 3-5 นาทีหลังโอน\nแล้วส่งสลิปใหม่" + "\n",
															Size:   "sm",
															Color:  "#ff0000",
															Weight: "bold",
														},
													},
												},
												&linebot.BoxComponent{
													Type:     "box",
													Layout:   "baseline",
													Spacing:  "sm",
													Margin:   "md",
													Contents: []linebot.FlexComponent{},
												},
												&linebot.BoxComponent{
													Type:        "box",
													Layout:      "horizontal",
													Spacing:     "sm",
													BorderWidth: "none",
													Contents: []linebot.FlexComponent{
														&linebot.BoxComponent{
															Type:            "box",
															Layout:          "vertical",
															CornerRadius:    "sm",
															BorderWidth:     "light",
															BorderColor:     "#ffc60a",
															AlignItems:      "center",
															BackgroundColor: "#ff0000",
															Contents: []linebot.FlexComponent{
																&linebot.TextComponent{
																	Type:  "text",
																	Text:  "ไปห้องเล่น",
																	Size:  "xxs",
																	Color: "#ffffff",
																	Wrap:  true,
																},
															},
															Action: actionP,
														},
														&linebot.BoxComponent{
															Type:            "box",
															Layout:          "vertical",
															CornerRadius:    "sm",
															BorderWidth:     "light",
															BorderColor:     "#ffc60a",
															AlignItems:      "center",
															BackgroundColor: "#00c000",
															Contents: []linebot.FlexComponent{
																&linebot.TextComponent{
																	Type:  "text",
																	Text:  "ไปห้องฝาก",
																	Size:  "xxs",
																	Color: "#ffffff",
																	Wrap:  true,
																},
															},
															Action: actionD,
														},
													},
												},
											},
										},
									},
								},
							},
						},
						Footer: &linebot.BoxComponent{
							Type:           "box",
							Layout:         "vertical",
							JustifyContent: "center",
							AlignItems:     "center",
							Contents:       []linebot.FlexComponent{},
							// Action: actionB,
						},
						Styles: &linebot.BubbleStyle{
							Header: &linebot.BlockStyle{
								BackgroundColor: "#ffc60a",
							},
							Body: &linebot.BlockStyle{
								BackgroundColor: "#000000",
							},
							Footer: &linebot.BlockStyle{
								BackgroundColor: "#ffc60a",
							},
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(event.ReplyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}
			} else if strings.Contains(result, "Bangkok") {
				// Handle the case where the response is "ABC"
				fmt.Println("Bangkok")
				flexMessage := linebot.NewFlexMessage(
					"เติมเงินไม่สำเร็จ",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "❌ สลิปจากธนาคารกรุงเทพรอ 3 นาทีหลังโอน แล้วส่งมาใหม่นะครับ",
									Weight: "bold",
									Size:   "md",
									Align:  "center",
									Color:  "#FF0000",
									Margin: "xs",
									Wrap:   true,
								},
								// &linebot.BoxComponent{
								// 	Type:   "box",
								// 	Layout: "horizontal",
								// 	Contents: []linebot.FlexComponent{
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  "ผู้รับเงิน:",
								// 			Size:  "xs",
								// 			Color: "#ffffff",
								// 		},
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  displayName,
								// 			Size:  "xs",
								// 			Color: "#ffffff",
								// 			Align: "end",
								// 		},
								// 	},
								// },

								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#ffffff",
									Margin: "sm",
								},
								&linebot.TextComponent{
									Type:   "text",
									Text:   "❌ ทำรายการไม่สำเร็จ",
									Weight: "bold",
									Size:   "md",
									Color:  "#FF0000",
									Margin: "xs",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#222222",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(event.ReplyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}
			} else if strings.Contains(result, "NOT QR") {
				fmt.Println("PICTURE")
			} else {
				// Process valid URLs or responses from the Python script
				fmt.Println("Valid response received:", result)
				if result == "0" {
					break
					fmt.Println("BREAK")
				}

				query := `
					UPDATE user_data 
					SET Credit = Credit + ? 
					WHERE ID = ? 
					`
				err = ExecuteQuery(ctx, query, result, userID)
				if err != nil {
					log.Printf("Error updating user data: %v", err)

				}

				// Check if any row was updated

				// Insert withdrawal log
				state := "ฝาก"
				if result[0] == '-' {
					state = "ถอน"
				}
				var idNum int // Declare the variable to store the result

				// Execute the query
				_ = db.QueryRowContext(ctx, `SELECT number FROM user_data WHERE id = ?`, userID).Scan(&idNum)

				wdLogQuery := `INSERT INTO wd (UID, STATE, NAME, AMOUNT, note) VALUES (?, ?, ?, ?, ?)`
				err = ExecuteQuery(ctx, wdLogQuery, idNum, state, displayName, result, "QR")
				if err != nil {
					log.Printf("Error inserting withdrawal log: %v", err)

				}

				// Fetch updated credit
				var credit float64
				err = db.QueryRowContext(ctx, `SELECT Credit FROM user_data WHERE Name = ?`, displayName).Scan(&credit)
				if err != nil {
					log.Printf("Error fetching updated credit: %v", err)

				}

				// // Prepare Flex message
				// flexMessage := linebot.NewFlexMessage(
				// 	"เติมเงินสำเร็จ",
				// 	&linebot.BubbleContainer{
				// 		Size: "kilo",
				// 		Body: &linebot.BoxComponent{
				// 			Type:   "box",
				// 			Layout: "vertical",
				// 			Contents: []linebot.FlexComponent{
				// 				&linebot.TextComponent{
				// 					Type:   "text",
				// 					Text:   "✅ ยืนยันการเติมเงิน",
				// 					Weight: "bold",
				// 					Size:   "md",
				// 					Color:  "#1DB446",
				// 					Margin: "xs",
				// 				},
				// 				&linebot.BoxComponent{
				// 					Type:   "box",
				// 					Layout: "horizontal",
				// 					Contents: []linebot.FlexComponent{
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  "ผู้รับเงิน:",
				// 							Size:  "xs",
				// 							Color: "#ffffff",
				// 						},
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  displayName,
				// 							Size:  "xs",
				// 							Color: "#ffffff",
				// 							Align: "end",
				// 						},
				// 					},
				// 				},
				// 				&linebot.BoxComponent{
				// 					Type:   "box",
				// 					Layout: "horizontal",
				// 					Contents: []linebot.FlexComponent{
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  "จำนวน:",
				// 							Size:  "xs",
				// 							Color: "#ffffff",
				// 						},
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  formatWithCommas(result),
				// 							Size:  "xs",
				// 							Color: "#ffffff",
				// 							Align: "end",
				// 						},
				// 					},
				// 				},
				// 				&linebot.BoxComponent{
				// 					Type:   "box",
				// 					Layout: "horizontal",
				// 					Contents: []linebot.FlexComponent{
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  "เครดิตรวม:",
				// 							Size:  "xs",
				// 							Color: "#ffffff",
				// 						},
				// 						&linebot.TextComponent{
				// 							Type:  "text",
				// 							Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)),
				// 							Size:  "xs",
				// 							Color: "#1DB446",
				// 							Align: "end",
				// 						},
				// 					},
				// 				},
				// 				&linebot.SeparatorComponent{
				// 					Type:   "separator",
				// 					Color:  "#ffffff",
				// 					Margin: "sm",
				// 				},
				// 				&linebot.TextComponent{
				// 					Type:   "text",
				// 					Text:   "✅ ทำรายการสำเร็จ",
				// 					Weight: "bold",
				// 					Size:   "md",
				// 					Color:  "#1DB446",
				// 					Margin: "xs",
				// 					Align:  "center",
				// 				},
				// 			},
				// 			Spacing:         "sm",
				// 			BackgroundColor: "#222222",
				// 		},
				// 	},
				// )
				// Prepare Flex message
				// actionB := NewClipboardAction("คัดลอกเลขบัญชี", BankAcc)
				actionP := linebot.NewURIAction("ไปห้องเล่น", PlayRoom)
				actionD := linebot.NewURIAction("ไปห้องฝาก", DepositRoom)
				flexMessage := linebot.NewFlexMessage(
					displayName+" เติมเงินผ่าน QR สำเร็จ",
					&linebot.BubbleContainer{
						Type: linebot.FlexContainerTypeBubble,
						Size: "deca",
						Header: &linebot.BoxComponent{
							Type:       linebot.FlexComponentTypeBox,
							Layout:     "vertical",
							Height:     "15px",
							Position:   "relative",
							AlignItems: "center",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:      linebot.FlexComponentTypeText,
									Text:      houseName,
									Size:      "xxs",
									Gravity:   "top",
									Align:     "start",
									Position:  "absolute",
									OffsetTop: "xs",
								},
							},
						},
						Body: &linebot.BoxComponent{
							Type:        linebot.FlexComponentTypeBox,
							Layout:      "vertical",
							PaddingAll:  "0px",
							BorderWidth: "normal",
							BorderColor: "#ffc60a",
							Contents: []linebot.FlexComponent{
								&linebot.BoxComponent{
									Type:     "box",
									Layout:   "horizontal",
									Contents: []linebot.FlexComponent{},
								},
								&linebot.BoxComponent{
									Type:       "box",
									Layout:     "horizontal",
									Spacing:    "xs",
									PaddingAll: "20px",
									Contents: []linebot.FlexComponent{
										&linebot.BoxComponent{
											Type:           "box",
											Layout:         "vertical",
											Width:          "55px",
											JustifyContent: "space-between",
											Contents: []linebot.FlexComponent{
												&linebot.BoxComponent{
													Type:         "box",
													Layout:       "vertical",
													CornerRadius: "100px",
													Width:        "48px",
													Height:       "48px",
													BorderWidth:  "medium",
													BorderColor:  "#ffc60a",
													Contents: []linebot.FlexComponent{
														&linebot.ImageComponent{
															Type:       "image",
															URL:        "https://thumb.r2.moele.me/t/24022/24012277/a-0145.jpg",
															AspectMode: "cover",
															Size:       "full",
														},
													},
												},
											},
										},
										&linebot.BoxComponent{
											Type:   "box",
											Layout: "vertical",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type: "text",
													Size: "sm",
													Wrap: true,
													Contents: []*linebot.SpanComponent{
														{
															Type:   "span",
															Text:   "ผู้ใช้ " + displayName + "\n",
															Weight: "bold",
															Color:  "#ffffff",
															Size:   "xxs",
														},
														{
															Type:   "span",
															Text:   "จำนวน " + formatWithCommas(result) + " ฿" + "\n",
															Size:   "md",
															Color:  "#ffffff",
															Weight: "bold",
														},
														{
															Type:  "span",
															Text:  "คงเหลือ " + formatWithCommas(fmt.Sprintf("%.0f", credit)) + " ฿",
															Color: "#1DB446",
														},
													},
												},
												&linebot.BoxComponent{
													Type:     "box",
													Layout:   "baseline",
													Spacing:  "sm",
													Margin:   "md",
													Contents: []linebot.FlexComponent{},
												},
												&linebot.BoxComponent{
													Type:        "box",
													Layout:      "horizontal",
													Spacing:     "sm",
													BorderWidth: "none",
													Contents: []linebot.FlexComponent{
														&linebot.BoxComponent{
															Type:            "box",
															Layout:          "vertical",
															CornerRadius:    "sm",
															BorderWidth:     "light",
															BorderColor:     "#ffc60a",
															AlignItems:      "center",
															BackgroundColor: "#ff0000",
															Contents: []linebot.FlexComponent{
																&linebot.TextComponent{
																	Type:  "text",
																	Text:  "ไปห้องเล่น",
																	Size:  "xxs",
																	Color: "#ffffff",
																	Wrap:  true,
																},
															},
															Action: actionP,
														},
														&linebot.BoxComponent{
															Type:            "box",
															Layout:          "vertical",
															CornerRadius:    "sm",
															BorderWidth:     "light",
															BorderColor:     "#ffc60a",
															AlignItems:      "center",
															BackgroundColor: "#00c000",
															Contents: []linebot.FlexComponent{
																&linebot.TextComponent{
																	Type:  "text",
																	Text:  "ไปห้องฝาก",
																	Size:  "xxs",
																	Color: "#ffffff",
																	Wrap:  true,
																},
															},
															Action: actionD,
														},
													},
												},
											},
										},
									},
								},
							},
						},
						Footer: &linebot.BoxComponent{
							Type:           "box",
							Layout:         "vertical",
							JustifyContent: "center",
							AlignItems:     "center",
							Contents:       []linebot.FlexComponent{},
							// Action: actionB,
						},
						Styles: &linebot.BubbleStyle{
							Header: &linebot.BlockStyle{
								BackgroundColor: "#ffc60a",
							},
							Body: &linebot.BlockStyle{
								BackgroundColor: "#000000",
							},
							Footer: &linebot.BlockStyle{
								BackgroundColor: "#ffc60a",
							},
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(event.ReplyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

			}
		}

	}
	// queueMutex.Lock() // Lock access to the queue
	// defer queueMutex.Unlock()
	for _, event := range body.Events {
		eventTimestamp := time.Unix(0, event.Timestamp*int64(time.Millisecond))
		userID := event.Source.UserID
		groupID := event.Source.GroupID
		sourceType := event.Source.Type
		log.Printf("Event from user: %s, source type: %s, ID Group: %s", userID, sourceType, groupID)
		if event.Message.Type == "sticker" && event.Source.Type == "group" {
			log.Printf("Sticker received! Package ID: %s, Sticker ID: %s", event.Message.PackageID, event.Message.StickerID)
			if event.Message.PackageID == "2" && event.Message.StickerID == "43" {
				event.Message.Type = "text"
				event.Message.Text = "E"
			}
		}
		// Handle text message
		// if event.Message.Type == "text" && event.Source.Type == "group" {
		if event.Message.Type == "text" {
			if translateNew2 {
				tempTemp2 := event.Message.Text
				tempTemp2 = strings.ReplaceAll(tempTemp2, " ", "")

				pattern1 := regexp.MustCompile(`^(ง|ด)(\d+)/(\d+)/(\d+)$`)
				pattern2 := regexp.MustCompile(`^(สง|สด)(\d+)/(\d+)$`)
				pattern3 := regexp.MustCompile(`^(ง|ด)(\d+)/(\d+)$`) // เพิ่ม pattern ใหม่

				// ตรวจสอบรูปแบบแรก: เช่น ง109/118/5000
				if matches := pattern1.FindStringSubmatch(tempTemp2); matches != nil {
					prefix := matches[1]
					num1 := matches[2]
					num2 := matches[3]
					amount := matches[4]
					event.Message.Text = fmt.Sprintf("%s %s %s = %s", prefix, num1, num2, amount)

				} else if matches := pattern2.FindStringSubmatch(tempTemp2); matches != nil {
					prefix := matches[1]
					num := matches[2]
					amount := matches[3]
					event.Message.Text = fmt.Sprintf("%s %s = %s", prefix, num, amount)

				} else if matches := pattern3.FindStringSubmatch(tempTemp2); matches != nil {
					prefix := matches[1]
					num := matches[2]
					amount := matches[3]
					event.Message.Text = fmt.Sprintf("%s %s รองได้ = %s", prefix, num, amount)
				}
				fmt.Println("AFTER 2:", event.Message.Text)
			}

			if translateNew {
				tempTemp := event.Message.Text

				event.Message.Text = strings.ReplaceAll(event.Message.Text, "(red circle)", "ด")  // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "(blue circle)", "ง") // Remove zero-width space🔴
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "🔵", "ง")
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "🔴", "ด")

				event.Message.Text = strings.ReplaceAll(event.Message.Text, "ดสด", "สด") // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "งสง", "สง") // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "งสง", "สง") // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "งสง", "สง") // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "รับ", "=")  // Remove zero-width space

				// Updated regex patterns
				// newpattern := `((?:สด|สง|ด|ง))\s*(\d+)(?:/(\d+)|\s+(\d+))?\s*=\s*([\d,]+)`
				newpattern := `((?:สด|สง|ด|ง))\s*(\d+)\s*(\d+)?\s*=\s*([\d,]+)`
				//((?:สด|สง|ด|ง))\s*(\d+)(?:/(\d+)|\s+(\d+))?\s*=\s*([\d,]+)

				pattern6 := `([ดง])\s*(\d+/?\d+)?\s*[\p{Thai}]*\s*=\s*(\d+)?`
				pattern7 := `([ดง])\s*(\d+\/\d+)?\s*(\d+\/\d+)?\s*=\s*([\d,]+)`

				numberOnlyPattern := `(\d+)\s+(\d+)\s*=\s*(\d+)` // Handles "10 10 = 1000"

				// Compile regex
				re := regexp.MustCompile(newpattern)
				re6 := regexp.MustCompile(pattern6)
				re7 := regexp.MustCompile(pattern7)
				reNumOnly := regexp.MustCompile(numberOnlyPattern)

				// Use event.Message.Text
				matches := re.FindStringSubmatch(event.Message.Text)
				matches6 := re6.FindStringSubmatch(event.Message.Text)
				matches7 := re7.FindStringSubmatch(event.Message.Text)

				matchesNumOnly := reNumOnly.FindStringSubmatch(event.Message.Text)

				// Predefined number mapping
				numberMap := map[string]string{
					"54": "5/4", "32": "3/2", "52": "5/2", "53": "5/3", "74": "7/4",
					"21": "2/1", "31": "3/1", "72": "7/2", "41": "4/1",
					"92": "9/2", "51": "5/1", "61": "6/1", "71": "7/1",
					"81": "8/1", "91": "9/1", "109": "10/9", "118": "11/8",
				}

				if matches != nil {
					symbol := matches[1] // "ด", "ง", "สด", "สง"
					num1 := matches[2]   // First number
					num2 := matches[3]   // Second number (optional)
					result := matches[4] // Result, may contain commas
					fmt.Println("ก่อนแปลง:", symbol, num1, num2, result)
					// Clean the result
					cleanedResult := strings.ReplaceAll(result, ",", "")

					// Convert num1
					if val, exists := numberMap[num1]; exists {
						num1 = val
					} else {
						num1 = num1 + "/1"
					}
					fmt.Println("num1 หลังแปลง:", num1)
					// Convert num2 (if exists)
					if num2 != "" {
						if val, exists := numberMap[num2]; exists {
							num2 = val
						} else {
							num2 = num2 + "/1"
						}
					}
					fmt.Println("num2 หลังแปลง:", num2)
					// Format result
					event.Message.Text = fmt.Sprintf("%s %s %s = %s\n", symbol, num1, num2, cleanedResult)
					fmt.Println("Final Message:", event.Message.Text)

				} else if matches7 != nil {
					symbol := matches7[1]   // ด / ง
					num := matches7[2]      // เช่น "52" หรือ "5/2"
					textThai := matches7[3] // "รองได้"
					result := matches7[4]   // เช่น "1000"

					fmt.Println("ก่อนแปลง7:", symbol, num, textThai, result)

					// แปลง num
					if val, exists := numberMap[num]; exists {
						num = val
					} else if !strings.Contains(num, "/") {
						num = num + "/1"
					}

					cleanedResult := strings.ReplaceAll(result, ",", "")
					event.Message.Text = fmt.Sprintf("%s %s %s = %s", symbol, num, textThai, cleanedResult)
					fmt.Println("Final Message7:", event.Message.Text)
				} else if matches6 != nil {
					// Handle pattern6 case
					num := matches6[2] // Number (e.g., "52" or "5/2")
					fmt.Println("ก่อนแปลง6:", num)
					text := event.Message.Text

					// Convert num if needed
					if val, exists := numberMap[num]; exists {
						num = val
					} else if !strings.Contains(num, "/") && !strings.Contains(num, "@") && num != "" && num != " " { // ตรวจสอบว่ามี "/" อยู่หรือยัง
						num = num + "/1"
					}
					fmt.Println("num3 หลังแปลง:", num)
					// Replace number in original text
					newText := strings.Replace(text, matches6[2], num, 1)
					event.Message.Text = newText

				} else if matchesNumOnly != nil {
					// Handle "10 10 = 1000" case (no symbols)
					num1 := matchesNumOnly[1]
					num2 := matchesNumOnly[2]
					result := matchesNumOnly[3]

					if num1 == num2 {
						num1 = num1 + "/" + num2
					} else {
						if val, exists := numberMap[num1]; exists {
							num1 = val
						} else {
							num1 = num1 + "/1"
						}

						if val, exists := numberMap[num2]; exists {
							num2 = val
						} else {
							num2 = num2 + "/1"
						}
					}

					// Format result
					fmt.Printf("%s = %s\n", num1, result)
					event.Message.Text = fmt.Sprintf("%s = %s\n", num1, result)

				} else {
					event.Message.Text = tempTemp
					fmt.Printf("No match: %s\n", event.Message.Text)
				}
			}
			if strings.Contains(event.Message.Text, "ตร") {
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "/", "=")  // Remove zero-width space
				event.Message.Text = strings.ReplaceAll(event.Message.Text, "109", "") // Remove zero-width space
			}
			fmt.Println("TRANSLATED,", event.Message.Text)
			pattern1 := `([ดง])\s*(\d+/\d+)\s*(\d+/\d+)\s*=\s*(\d+)`
			pattern2 := `([ดง])\s*(\d+/\d+)\s*รอง\p{Thai}+\s*=\s*(\d+)`
			pattern2N := `([ดง])\s*(\d+/\d+)\s*ต่อ\p{Thai}+\s*=\s*(\d+)`
			pattern3 := `(ส[ด|ง])\s*(\d+/\d+)\s*=\s*(\d+)`
			pattern4 := `(ตร)\s*(\d+/\d+)?\s*=\s*(\d+)`
			pattern5 := `10/10\s*=\s*(\d{2,7})`
			pattern6 := `([ดงสตร]?)\s*(\d+/?\d+)?\s*[\p{Thai}]*\s*=\s*(\d+)?`
			// pattern := `^([ดตง])[\n/\\]?(\d+)$`

			// Define the special message texts
			specialMessages := []string{"ปิด", "ปด", "ป", "E", "(prohibited)", "(x mark)"}

			// Check if the message text matches any of the special messages
			matchesSpecial := contains(specialMessages, event.Message.Text)

			// Check if the message text matches any of the patterns
			matchesPattern := false
			for _, p := range []string{pattern1, pattern2, pattern2N, pattern3, pattern4, pattern5, pattern6} {
				re := regexp.MustCompile(p)
				if re.MatchString(event.Message.Text) {
					matchesPattern = true
					break
				}
			}
			// Check if the message text matches any of the special messages
			if matchesSpecial || matchesPattern || true {
				if playON {
					replyMessage := fmt.Sprintf("You sent: \"%s\" at %s", event.Message.Text, eventTimestamp)
					heap.Push(eventQueue, &Request{
						ReplyToken:   event.ReplyToken,
						RawMessage:   event.Message.Text,
						ReplyMessage: replyMessage,
						QuoteToken:   event.Message.QuoteToken,
						Timestamp:    eventTimestamp,
						UID:          userID,
						GroupID:      groupID,
						Matched:      matchesSpecial || matchesPattern,
						Sequence:     atomic.AddInt64(&sequenceCounter, 1), // เพิ่ม sequence
					})

				}

				// Check if the message text matches one of the patterns

			} else {
				replyMessage := fmt.Sprintf("You sent: \"%s\" at %s", event.Message.Text, eventTimestamp)
				if err := ReplyToLine(event.ReplyToken, event.Message.Text, replyMessage, userID, groupID, event.Message.QuoteToken); err != nil {
					log.Printf("Error sending reply: %v", err)
				}
			}
			// replyMessage := fmt.Sprintf("You sent: \"%s\" at %s", event.Message.Text, eventTimestamp)
			// if err := ReplyToLine(event.ReplyToken, event.Message.Text, replyMessage, userID, groupID, event.Message.QuoteToken); err != nil {
			// 	log.Printf("Error sending reply: %v", err)
			// } else {
			// 	replyMessage := fmt.Sprintf("You sent: \"%s\" at %s", event.Message.Text, eventTimestamp)
			// 	heap.Push(eventQueue, &Request{
			// 		ReplyToken:   event.ReplyToken,
			// 		RawMessage:   event.Message.Text,
			// 		ReplyMessage: replyMessage,
			// 		QuoteToken:   event.Message.QuoteToken,
			// 		Timestamp:    eventTimestamp,
			// 		UID:          userID,
			// 		GroupID:      groupID,
			// 	})
			// }

		}

	}

	// Handle image message

	w.WriteHeader(http.StatusOK)
	go processQueue() // Start processing in a separate goroutine
}
func scanQRCode(imagePath string) (string, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	// Decode the image to get the raw pixel data
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Encode the image to a buffer
	var buf bytes.Buffer
	err = encodeImageToBuffer(img, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to encode image to buffer: %w", err)
	}

	// Now we have a byte slice in buf, we can decode it using qrcode.Decode
	qrCode, err := qrcode.Decode(&buf)
	if err != nil {
		return "", fmt.Errorf("failed to decode QR code: %w", err)
	}

	// Return the decoded content from the QR code
	return qrCode.Content, nil
}

// Helper function to encode an image to a buffer (e.g., PNG format)
func encodeImageToBuffer(img image.Image, buf *bytes.Buffer) error {
	// You can choose to encode in PNG, JPEG, etc.
	// Here we use PNG encoding
	err := png.Encode(buf, img)
	return err
}
func processQueue() {
	if isProcessing {
		return
	}

	isProcessing = true
	defer func() { isProcessing = false }()

	for len(*eventQueue) > 0 {
		if len(*eventQueue) < 4 {
			time.Sleep(464 * time.Millisecond)

			// time.Sleep(0 * time.Millisecond)
		}
		queueMutex.Lock()

		// Lock access to the queue
		heap.Init(eventQueue)
		req := heap.Pop(eventQueue).(*Request)

		if err := ReplyToLine(req.ReplyToken, req.RawMessage, req.ReplyMessage, req.UID, req.GroupID, req.QuoteToken); err != nil {
			log.Printf("Error sending reply: %v", err)
		}
		queueMutex.Unlock() // Unlock after the operation
		log.Printf("Processed request: %v", req)
		time.Sleep(5 * time.Millisecond)
	}
}

// func processQueue() {
// 	if isProcessing {
// 		return
// 	}

// 	isProcessing = true
// 	defer func() { isProcessing = false }()

// 	ticker := time.NewTicker(300 * time.Millisecond)
// 	defer ticker.Stop()

// 	var lastProcessedTimestamp time.Time
// 	var lastQueueSize int

// 	for {
// 		select {
// 		case <-ticker.C:
// 			queueMutex.Lock()
// 			currentQueueSize := len(*eventQueue)

// 			// ถ้าคิวเหลือแค่ 1 ให้ใช้เงื่อนไขเดิม (รอให้คิวมีเสถียรภาพ)
// 			if currentQueueSize == 2 && currentQueueSize > lastQueueSize {
// 				lastQueueSize = currentQueueSize
// 				queueMutex.Unlock()
// 				continue
// 			}
// 			if currentQueueSize == 1 && currentQueueSize > lastQueueSize {
// 				lastQueueSize = currentQueueSize
// 				queueMutex.Unlock()
// 				continue
// 			}

// 			if currentQueueSize == 0 {
// 				queueMutex.Unlock()
// 				continue
// 			}

// 			// อัปเดต timestamp ที่กำลังประมวลผล
// 			if lastProcessedTimestamp.IsZero() || (*eventQueue)[0].Timestamp.After(lastProcessedTimestamp) {
// 				lastProcessedTimestamp = (*eventQueue)[0].Timestamp
// 			}

// 			// ดึงทุก Request ออกจากคิวแบบปลอดภัย ยกเว้นรายการสุดท้าย
// 			var batch []*Request
// 			var specialBatch []*Request
// 			if len(*eventQueue) > 2 {
// 				for len(*eventQueue) > 2 {
// 					req := heap.Pop(eventQueue).(*Request)
// 					if req.Matched {
// 						specialBatch = append(specialBatch, req)
// 					} else {
// 						batch = append(batch, req)
// 					}
// 					if req.Matched {
// 						break
// 					}
// 				}
// 			} else if len(*eventQueue) > 1 {
// 				for len(*eventQueue) > 1 {
// 					req := heap.Pop(eventQueue).(*Request)
// 					if req.Matched {
// 						specialBatch = append(specialBatch, req)
// 					} else {
// 						batch = append(batch, req)
// 					}
// 					if req.Matched {
// 						break
// 					}
// 				}
// 			} else if len(*eventQueue) > 0 {
// 				for len(*eventQueue) > 0 {
// 					req := heap.Pop(eventQueue).(*Request)
// 					if req.Matched {
// 						specialBatch = append(specialBatch, req)
// 					} else {
// 						batch = append(batch, req)
// 					}
// 					if req.Matched {
// 						break
// 					}
// 				}
// 			}

// 			// อัปเดตขนาดคิวล่าสุด
// 			lastQueueSize = len(*eventQueue)
// 			queueMutex.Unlock()

// 			var wg sync.WaitGroup

// 			// ประมวลผล batch ปกติ
// 			if len(batch) > 0 {
// 				wg.Add(1)
// 				go func() {
// 					defer wg.Done()
// 					processBatchParallel(batch)
// 				}()
// 			}

// 			wg.Wait()

// 			// ประมวลผล specialBatch
// 			if len(specialBatch) > 0 {
// 				wg.Add(1)
// 				go func() {
// 					defer wg.Done()
// 					processBatchParallel(specialBatch)
// 				}()
// 			}

// 			wg.Wait()
// 		}
// 	}
// }

// // ฟังก์ชันประมวลผลแบบ Multi-threaded
// func processBatchParallel(requests []*Request) {
// 	const maxGoroutines = 1
// 	semaphore := make(chan struct{}, maxGoroutines)

// 	var wg sync.WaitGroup

// 	for _, req := range requests {
// 		wg.Add(1)

// 		semaphore <- struct{}{}
// 		go func(r *Request) {
// 			defer wg.Done()
// 			defer func() { <-semaphore }()

// 			if err := ReplyToLine(r.ReplyToken, r.RawMessage, r.ReplyMessage, r.UID, r.GroupID, r.QuoteToken); err != nil {
// 				log.Printf("Error sending reply: %v", err)
// 			} else {
// 				log.Printf("Processed request: %v", r)
// 			}
// 		}(req)
// 	}

// 	wg.Wait()
// 	log.Printf("Processed batch of %d requests", len(requests))
// }

func SendFlexMessageBank(replyToken string) error {
	// Define the Flex message with clipboard action as a JSON string
	flexMessage := `{
		"type": "flex",
		"altText": "Flex Message",
		"contents": {
			"type": "bubble",
			"body": {
				"type": "box",
				"layout": "vertical",
				"contents": [
					{
						"type": "button",
						"action": {
							"type": "clipboard",
							"label": "คัดลอก",
							"clipboardText": "ข้อความคัดลอก"
						},
						"style": "primary",
						"color": "#0000ff"
					}
				]
			}
		}
	}`

	// Create the payload to send the Flex message
	payload := fmt.Sprintf(`{
		"replyToken": "%s",
		"messages": [%s]
	}`, replyToken, flexMessage)

	// Send the request to LINE's messaging API
	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/reply", bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+LineChannelAccessToken)

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE API error: %v", resp.Status)
	}

	fmt.Println("Flex message with clipboard action sent successfully!")
	return nil
}

// เพิ่ม ActionTypeClipboard ลงใน ActionType constants

const (
	ActionTypeClipboard linebot.ActionType = "clipboard"
)

// ClipboardAction type
type ClipboardAction struct {
	Label         string `json:"label"`
	ClipboardText string `json:"clipboardText"`
}

// MarshalJSON method of ClipboardAction
func (a *ClipboardAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type          linebot.ActionType `json:"type"`
		Label         string             `json:"label"`
		ClipboardText string             `json:"clipboardText"`
	}{
		Type:          "clipboard",
		Label:         a.Label,
		ClipboardText: a.ClipboardText,
	})
}

// TemplateAction implements TemplateAction interface
func (*ClipboardAction) TemplateAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*ClipboardAction) QuickReplyAction() {}

// NewClipboardAction function
func NewClipboardAction(label, text string) *ClipboardAction {
	return &ClipboardAction{
		Label:         label,
		ClipboardText: text,
	}
}

type BankAccount struct {
	BankName      string
	BankNameEn    string
	AccountNumber string
	AccountName   string
	AccountNameEn string
	LogoURL       string
}

func GetBankAccount() (*BankAccount, error) {
	dsn := "duckcom_fulloption2:duckcom_fulloption2@tcp(203.170.129.1:3306)/duckcom_fulloption2"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	query := "SELECT bank_name, bank_name_en, account_number, account_name, account_name_en, logo_url FROM bank_accounts WHERE id = 1"
	row := db.QueryRow(query)

	var ba BankAccount
	err = row.Scan(&ba.BankName, &ba.BankNameEn, &ba.AccountNumber, &ba.AccountName, &ba.AccountNameEn, &ba.LogoURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no record found with id = 1")
		}
		return nil, fmt.Errorf("failed to scan row: %v", err)
	}

	return &ba, nil
}

func ReplyToLine(replyToken, rawMessage, replyMessage, UID, GroupID, quoteToken string) error {
	// Send the message with the quoteToken included if available
	pattern1 := `([ดง])\s*(\d+/\d+)\s*(\d+/\d+)\s*=\s*(\d+)`
	pattern2 := `([ดง])\s*(\d+/\d+)\s*รอง\p{Thai}+\s*=\s*(\d+)`
	pattern2N := `([ดง])\s*(\d+/\d+)\s*ต่อ\p{Thai}+\s*=\s*(\d+)`
	pattern3 := `(ส[ด|ง])\s*(\d+/\d+)\s*=\s*(\d+)`
	pattern4 := `(ตร)\s*(\d+/\d+)?\s*=\s*(\d+)`
	pattern5 := `10/10\s*=\s*(\d{2,7})`
	pattern6 := `([ดงสตร]?)\s*(\d+/?\d+)?\s*[\p{Thai}]*\s*=\s*(\d+)?`

	// Define the special message texts
	specialMessages := []string{"ปิด", "ปด", "ป", "E", "(prohibited)", "(x mark)"}

	// Check if the message text matches any of the special messages
	matchesSpecial := contains(specialMessages, rawMessage)

	// Check if the message text matches any of the patterns
	matchesPattern := false
	for _, p := range []string{pattern1, pattern2, pattern2N, pattern3, pattern4, pattern5, pattern6} {
		re := regexp.MustCompile(p)
		if re.MatchString(rawMessage) {
			matchesPattern = true
			break
		}
	}
	// Check if the message text matches any of the special messages
	if matchesSpecial || matchesPattern {
		quoteToken = ""
	}
	bankAcc, _ := GetBankAccount()
	actionB := NewClipboardAction("คัดลอกเลขบัญชี", bankAcc.AccountNumber)
	actionP := linebot.NewURIAction("ไปห้องเล่น", PlayRoom)
	actionD := linebot.NewURIAction("ไปห้องฝาก", DepositRoom)
	message := MessagePayload{Type: "text", Text: CheckMessage(rawMessage, UID, GroupID, replyToken), QuoteToken: quoteToken}
	// message := MessagePayload{Type: "text", Text: replyMessage, QuoteToken: quoteToken}
	// flexMessageOLD := linebot.NewFlexMessage(
	// 	"เลขบัญชี",
	// 	&linebot.BubbleContainer{
	// 		Size: "kilo", // Size of the bubble
	// 		Hero: &linebot.BoxComponent{
	// 			Type:   "box",      // Box component for the body
	// 			Layout: "vertical", // Vertical layout
	// 			Contents: []linebot.FlexComponent{
	// 				&linebot.ImageComponent{
	// 					Type:        "image", // Type of component is image
	// 					URL:         BankURL, // Image URL
	// 					Size:        "full",  // Full size image
	// 					AspectRatio: "23:9",  // Aspect ratio
	// 					AspectMode:  "cover", // Aspect mode
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",                        // Text component
	// 					Text:   fmt.Sprintf("(%v)", BankName), // Text to display
	// 					Weight: "bold",                        // Bold text
	// 					Size:   "md",                          // Medium size text
	// 					Align:  "center",
	// 					Margin: "10px",
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",                       // Text component
	// 					Text:   fmt.Sprintf("(%v)", BankAcc), // Text to display
	// 					Weight: "bold",                       // Bold text
	// 					Size:   "xxl",                        // Medium size text
	// 					Align:  "center",``
	// 					Color:  "#FF0000",
	// 					Margin: "10px",
	// 					Action: actionB,
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",      // Text component
	// 					Text:   UserAccName, // Text to display
	// 					Weight: "bold",      // Bold text
	// 					Size:   "md",        // Medium size text
	// 					Align:  "center",
	// 					Margin: "10px",
	// 				},
	// 			},
	// 		},
	// 		Body: &linebot.BoxComponent{
	// 			Type:            "box",      // Box component for the body
	// 			Layout:          "vertical", // Vertical layout
	// 			Action:          actionB,
	// 			BackgroundColor: "#FFFFFF",
	// 			PaddingAll:      "md",

	// 			Contents: []linebot.FlexComponent{
	// 				&linebot.BoxComponent{
	// 					Type:            "box",      // Box component for the body
	// 					Layout:          "vertical", // Vertical layout
	// 					Action:          actionB,
	// 					BackgroundColor: "#32CD32",
	// 					PaddingAll:      "md",
	// 					CornerRadius:    "xxl",

	// 					Contents: []linebot.FlexComponent{
	// 						&linebot.TextComponent{
	// 							Type:   "text",                            // Text component
	// 							Text:   "(คลิกที่นี่เพื่อคัดลอกเลขบัญชี)", // Text to display
	// 							Weight: "bold",                            // Bold text
	// 							Size:   "sm",                              // Medium size text
	// 							Align:  "center",

	// 							Color: "#FFFFFF",
	// 						},
	// 					},
	// 				},

	// 				// &linebot.TextComponent{
	// 				// 	Type:   "text",                                                                                                      // Text component
	// 				// 	Text:   fmt.Sprintf("(เมื่อโอนสำเร็จแล้วสามารถส่งสลิปที่ห้องฝากได้เลยครับ ธนาคารกรุงเทพ รอ 5 นาทีก่อนส่งนะครับ^^)"), // Text to display
	// 				// 	Weight: "bold",                                                                                                      // Bold text
	// 				// 	Size:   "sm",                                                                                                        // Medium size text
	// 				// 	Align:  "center",
	// 				// 	Wrap:   true,
	// 				// },
	// 			},
	// 		},
	// 	},
	// )

	flexMessage := linebot.NewFlexMessage(
		"เลขบัญชี",
		&linebot.BubbleContainer{
			Type: linebot.FlexContainerTypeBubble,
			Size: "deca",
			Header: &linebot.BoxComponent{
				Type:       linebot.FlexComponentTypeBox,
				Layout:     "vertical",
				Height:     "15px",
				Position:   "relative",
				AlignItems: "center",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:      linebot.FlexComponentTypeText,
						Text:      houseName,
						Size:      "xxs",
						Gravity:   "top",
						Align:     "start",
						Position:  "absolute",
						OffsetTop: "xs",
					},
				},
			},
			Body: &linebot.BoxComponent{
				Type:        linebot.FlexComponentTypeBox,
				Layout:      "vertical",
				PaddingAll:  "0px",
				BorderWidth: "normal",
				BorderColor: "#ffc60a",
				Contents: []linebot.FlexComponent{
					&linebot.BoxComponent{
						Type:     "box",
						Layout:   "horizontal",
						Contents: []linebot.FlexComponent{},
					},
					&linebot.BoxComponent{
						Type:       "box",
						Layout:     "horizontal",
						Spacing:    "xs",
						PaddingAll: "20px",
						Contents: []linebot.FlexComponent{
							&linebot.BoxComponent{
								Type:           "box",
								Layout:         "vertical",
								Width:          "55px",
								JustifyContent: "space-between",
								Contents: []linebot.FlexComponent{
									&linebot.BoxComponent{
										Type:         "box",
										Layout:       "vertical",
										CornerRadius: "100px",
										Width:        "48px",
										Height:       "48px",
										BorderWidth:  "medium",
										BorderColor:  "#ffc60a",
										Contents: []linebot.FlexComponent{
											&linebot.ImageComponent{
												Type:       "image",
												URL:        bankAcc.LogoURL,
												AspectMode: "cover",
												Size:       "full",
											},
										},
									},
								},
							},
							&linebot.BoxComponent{
								Type:   "box",
								Layout: "vertical",
								Contents: []linebot.FlexComponent{
									&linebot.TextComponent{
										Type: "text",
										Size: "sm",
										Wrap: true,
										Contents: []*linebot.SpanComponent{
											{
												Type:   "span",
												Text:   bankAcc.BankName + "\n",
												Weight: "bold",
												Color:  "#ffffff",
												Size:   "xxs",
											},
											{
												Type:   "span",
												Text:   bankAcc.AccountNumber + "\n",
												Size:   "md",
												Color:  "#00ff00",
												Weight: "bold",
											},
											{
												Type:  "span",
												Text:  bankAcc.AccountName,
												Color: "#ffffff",
											},
										},
									},
									&linebot.BoxComponent{
										Type:     "box",
										Layout:   "baseline",
										Spacing:  "sm",
										Margin:   "md",
										Contents: []linebot.FlexComponent{},
									},
									&linebot.BoxComponent{
										Type:        "box",
										Layout:      "horizontal",
										Spacing:     "sm",
										BorderWidth: "none",
										Contents: []linebot.FlexComponent{
											&linebot.BoxComponent{
												Type:            "box",
												Layout:          "vertical",
												CornerRadius:    "sm",
												BorderWidth:     "light",
												BorderColor:     "#ffc60a",
												AlignItems:      "center",
												BackgroundColor: "#ff0000",
												Contents: []linebot.FlexComponent{
													&linebot.TextComponent{
														Type:  "text",
														Text:  "ไปห้องเล่น",
														Size:  "xxs",
														Color: "#ffffff",
														Wrap:  true,
													},
												},
												Action: actionP,
											},
											&linebot.BoxComponent{
												Type:            "box",
												Layout:          "vertical",
												CornerRadius:    "sm",
												BorderWidth:     "light",
												BorderColor:     "#ffc60a",
												AlignItems:      "center",
												BackgroundColor: "#00c000",
												Contents: []linebot.FlexComponent{
													&linebot.TextComponent{
														Type:  "text",
														Text:  "ไปห้องฝาก",
														Size:  "xxs",
														Color: "#ffffff",
														Wrap:  true,
													},
												},
												Action: actionD,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Footer: &linebot.BoxComponent{
				Type:           "box",
				Layout:         "vertical",
				JustifyContent: "center",
				AlignItems:     "center",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:  "text",
						Text:  "คัดลอกเลขบัญชี",
						Size:  "xs",
						Color: "#000000",
					},
				},
				Action: actionB,
			},
			Styles: &linebot.BubbleStyle{
				Header: &linebot.BlockStyle{
					BackgroundColor: "#ffc60a",
				},
				Body: &linebot.BlockStyle{
					BackgroundColor: "#000000",
				},
				Footer: &linebot.BlockStyle{
					BackgroundColor: "#ffc60a",
				},
			},
		},
	)
	flexRules := linebot.NewFlexMessage(
		"กติกา",
		&linebot.CarouselContainer{
			Type: linebot.FlexContainerTypeCarousel,
			Contents: []*linebot.BubbleContainer{
				{
					Type: linebot.FlexContainerTypeBubble,
					Size: "kilo",
					Body: &linebot.BoxComponent{
						Type:       linebot.FlexComponentTypeBox,
						Layout:     "vertical",
						Spacing:    "none",
						PaddingAll: "0px",
						Contents: []linebot.FlexComponent{
							&linebot.ImageComponent{
								Type: linebot.FlexComponentTypeImage,
								URL:  "https://i.postimg.cc/RZjykV51/S-3244119-0.jpg",
								Size: "full",
							},
						},
					},
				},
				{
					Type: linebot.FlexContainerTypeBubble,
					Size: "kilo",
					Body: &linebot.BoxComponent{
						Type:       linebot.FlexComponentTypeBox,
						Layout:     "vertical",
						Spacing:    "none",
						PaddingAll: "0px",
						Contents: []linebot.FlexComponent{
							&linebot.ImageComponent{
								Type: linebot.FlexComponentTypeImage,
								URL:  "https://i.postimg.cc/zfK68Z8s/S-3244121-0.jpg",
								Size: "full",
							},
						},
					},
				},
			},
		},
	)

	// flexMessage2 := linebot.NewFlexMessage(
	// 	"Flex message with hero",
	// 	&linebot.BubbleContainer{
	// 		Size: "kilo", // Size of the bubble
	// 		Hero: &linebot.BoxComponent{
	// 			Type:   "box",      // Box component for the body
	// 			Layout: "vertical", // Vertical layout
	// 			Contents: []linebot.FlexComponent{
	// 				&linebot.ImageComponent{
	// 					Type:        "image",  // Type of component is image
	// 					URL:         BankURL2, // Image URL
	// 					Size:        "full",   // Full size image
	// 					AspectRatio: "23:9",   // Aspect ratio
	// 					AspectMode:  "cover",  // Aspect mode
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",                         // Text component
	// 					Text:   fmt.Sprintf("(%v)", BankName2), // Text to display
	// 					Weight: "bold",                         // Bold text
	// 					Size:   "md",                           // Medium size text
	// 					Align:  "center",
	// 					Margin: "10px",
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",                        // Text component
	// 					Text:   fmt.Sprintf("(%v)", BankAcc2), // Text to display
	// 					Weight: "bold",                        // Bold text
	// 					Size:   "md",                          // Medium size text
	// 					Align:  "center",
	// 					Margin: "10px",
	// 				},
	// 				&linebot.TextComponent{
	// 					Type:   "text",       // Text component
	// 					Text:   UserAccName2, // Text to display
	// 					Weight: "bold",       // Bold text
	// 					Size:   "md",         // Medium size text
	// 					Align:  "center",
	// 					Margin: "10px",
	// 				},
	// 			},
	// 		},
	// 		Body: &linebot.BoxComponent{
	// 			Type:   "box",      // Box component for the body
	// 			Layout: "vertical", // Vertical layout
	// 			Contents: []linebot.FlexComponent{

	// 				&linebot.TextComponent{
	// 					Type:   "text",                                        // Text component
	// 					Text:   fmt.Sprintf("(สามารถคัดลอกเลขบัญชีด้านล่าง)"), // Text to display
	// 					Weight: "bold",                                        // Bold text
	// 					Size:   "md",                                          // Medium size text
	// 					Align:  "center",
	// 				},
	// 				// &linebot.TextComponent{
	// 				// 	Type:   "text",                                                                                                      // Text component
	// 				// 	Text:   fmt.Sprintf("(เมื่อโอนสำเร็จแล้วสามารถส่งสลิปที่ห้องฝากได้เลยครับ ธนาคารกรุงเทพ รอ 5 นาทีก่อนส่งนะครับ^^)"), // Text to display
	// 				// 	Weight: "bold",                                                                                                      // Bold text
	// 				// 	Size:   "sm",                                                                                                        // Medium size text
	// 				// 	Align:  "center",
	// 				// 	Wrap:   true,
	// 				// },
	// 			},
	// 		},
	// 	},
	// )
	// flexMessage := linebot.NewFlexMessage(
	// 	"Flex message with clipboard action", // Alt text
	// 	&linebot.BubbleContainer{
	// 		Size: "kilo", // Size of the bubble
	// 		Body: &linebot.BoxComponent{
	// 			Type:   "box",      // Box component for the body
	// 			Layout: "vertical", // Vertical layout
	// 			Contents: []linebot.FlexComponent{
	// 				&linebot.ButtonComponent{
	// 					Type: "button", // Button type
	// 					Action: &linebot.ClipboardAction{
	// 						Label:        "คัดลอก",               // Button label
	// 						ClipboardText: "ข้อความคัดลอก",      // Text to copy to clipboard
	// 					},
	// 					Style: "primary", // Button style
	// 					Color: "#0000ff", // Button color
	// 				},
	// 			},
	// 		},
	// 	},
	// )

	// // Send the Flex message using the bot
	// _, err := bot.ReplyMessage(replyToken, flexMessage).Do()

	// Create regular text message
	// message5 := linebot.NewTextMessage(BankAcc)
	// message52 := linebot.NewTextMessage(BankAcc2)
	var messages []linebot.SendingMessage
	var messages2 []linebot.SendingMessage

	messages = append(messages, flexMessage)
	// messages = append(messages, message5)
	messages2 = append(messages2, flexRules)
	// messages = append(messages, flexMessage2)
	// messages = append(messages, message52)

	// Wrap Flex message in MessagePayload
	message2 := MessagePayload{
		Type:        "flex",      // Flex type
		FlexMessage: flexMessage, // The Flex message content
		QuoteToken:  quoteToken,  // Optional: include the quote token if needed
	}

	// message2 := MessagePayload{Type: "text", Text: replyMessage, QuoteToken: quoteToken}
	message3 := MessagePayload{Type: "text", Text: replyMessage}
	payload := map[string]interface{}{
		"replyToken": replyToken,
		"messages":   []MessagePayload{message},
	}
	if rawMessage == "บช2" {
		fmt.Println("OK11")
		payload = map[string]interface{}{
			"replyToken": replyToken,
			"messages":   []MessagePayload{message2, message3},
		}
	} else if rawMessage == "บช" {
		bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
		if _, err := bot.ReplyMessage(replyToken, messages...).Do(); err != nil {
			log.Println(err)
		}
	} else if rawMessage == "กติกาXXXXx" {
		bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
		if _, err := bot.ReplyMessage(replyToken, messages2...).Do(); err != nil {
			log.Println(err)
		}
	}

	return sendLineReply(payload)
}

func sendLineReply(payload map[string]interface{}) error {
	url := "https://api.line.me/v2/bot/message/reply"
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+LineChannelAccessToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Printf("Reply sent successfully: %s", jsonData)
	return nil
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Fetch event history if needed, e.g., all processed requests
	json.NewEncoder(w).Encode(eventQueue)
}

func main() {

	InitDB()
	// log.Default().SetOutput(ioutil.Discard)
	heap.Init(eventQueue)
	http.HandleFunc("/botoption2", WebhookHandler)
	http.HandleFunc("/history", HistoryHandler)
	log.Printf("Server running on http://localhost:5065")
	log.Println(http.ListenAndServe(":5065", nil))
}

// /DB
var db *sql.DB
var once sync.Once
var lineBotAPI *linebot.Client

// InitDB initializes the database connection pool
func InitDB() {
	once.Do(func() {
		var err error
		dsn := "duckcom_fulloption2:duckcom_fulloption2@tcp(203.170.129.1:3306)/duckcom_fulloption2"
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Println("Error connecting to database:", err)
		}
		lineBotAPI, _ = linebot.New(LineChannelSecret, LineChannelAccessToken)

		// Set connection pool settings
		db.SetMaxOpenConns(100)  // Maximum number of open connections
		db.SetMaxIdleConns(40)   // Maximum number of idle connections
		db.SetConnMaxLifetime(0) // Connections never expire

		// Verify the connection
		if err := db.Ping(); err != nil {
			log.Println("Error pinging database:", err)
		}
		log.Println("Database connection pool initialized")
	})
}

// ExecuteQuery executes a given SQL query and closes the connection
func ExecuteQuery(ctx context.Context, query string, args ...interface{}) error {
	InitDB() // Ensure the database connection pool is initialized

	// Use a connection from the pool
	tx, err := db.BeginTx(ctx, nil) // Begin a new transaction
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		tx.Rollback() // Rollback on error
		return fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		tx.Rollback() // Rollback on error
		return fmt.Errorf("error executing query: %v", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}
func getTeamSumsAndTotalDeposit(round int) (int, int, int, int, error) {
	// Connect to the database
	dsn := "duckcom_fulloption2:duckcom_fulloption2@tcp(203.170.129.1:3306)/duckcom_fulloption2"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	// SQL query to calculate total deposit from 'wd' table
	sqlQueryDeposit := `SELECT SUM(AMOUNT) as totalSum FROM wd WHERE  checks = 0 AND (STATE = 'ฝาก' OR STATE = 'ถอน')`
	var totalDeposit int
	_ = db.QueryRow(sqlQueryDeposit).Scan(&totalDeposit)

	// SQL query to calculate total sums for red and blue teams for the round
	sqlQueryTeamSums := `
		SELECT 
			-1 * (SUM(CASE WHEN b1 > 0 THEN maxprofit ELSE 0 END) - 
				  SUM(CASE WHEN j1 > 0 THEN maxlost ELSE 0 END)) AS b1_sum,
			-1 * (SUM(CASE WHEN j1 > 0 THEN maxprofit ELSE 0 END) - 
				  SUM(CASE WHEN b1 > 0 THEN maxlost ELSE 0 END)) AS j1_sum
		FROM playinglog
		WHERE Game_play != 'END' 
			AND Game_play != 'AFTER' 
			AND round = ?;
	`
	var b1Sum, j1Sum int
	err = db.QueryRow(sqlQueryTeamSums, round).Scan(&b1Sum, &j1Sum)
	if err != nil {
		if err == sql.ErrNoRows {
			b1Sum, j1Sum = 0, 0
		}
	}

	// SQL query to calculate the grand total
	sqlQueryGrandTotal := `
		SELECT 
			-1*SUM(pl.balance) as RoundSum
		FROM user_data p
		LEFT JOIN (
			SELECT DISTINCT ID, round, balance
			FROM playinglog
			WHERE Game_play != 'END'
		) pl ON p.ID = pl.ID
		WHERE EXISTS (
			SELECT 1
			FROM playinglog pl_check
			WHERE pl_check.ID = p.ID
			  AND pl_check.Game_play != 'END'
		)
		GROUP BY p.ID, pl.round
		ORDER BY p.ID, pl.round ASC;
	`
	rows, _ := db.Query(sqlQueryGrandTotal)
	// if err != nil {
	// 	return 0, 0, 0, 0, fmt.Errorf("failed to execute grand total query: %v", err)
	// }
	defer rows.Close()

	// Process results to calculate grand total
	var grandTotal int
	for rows.Next() {
		var roundSum int
		err := rows.Scan(&roundSum)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("failed to scan grand total row: %v", err)
		}
		grandTotal += roundSum
	}

	// If no rows were processed, grandTotal will remain 0
	return totalDeposit, grandTotal, b1Sum, j1Sum, nil
}
func Reverse(ctx context.Context, state int, round int) (string, error) {
	if state != 0 {
		return "ติดต่อผู้ให้บริการเพื่อดำเนินการแก้ไข", nil
	}
	// round = round - 1
	// Fetch results from playinglog
	fmt.Println("1PAST")
	query := `SELECT DISTINCT ID, balance, adminSum
          FROM playinglog WHERE round = ? AND Game_play != ? AND Game_play != ?`
	fmt.Printf("Executing query: %s with round=%d, Game_play1=%s, Game_play2=%s\n", query, round-1, "END", "AFTER")

	rows, err := db.QueryContext(ctx, query, round-1, "END", "AFTER")
	if err != nil {
		return "", fmt.Errorf("error fetching results: %w", err)
	}
	defer rows.Close()

	// Debug: Count rows fetched
	rowCount := 0
	fmt.Println("Query executed successfully, processing rows...")
	fmt.Println("2PAST")
	// Process each row
	for rows.Next() {
		var id string
		var balance, adminSum sql.NullFloat64

		fmt.Println("Attempting to scan row...")
		if err := rows.Scan(&id, &balance, &adminSum); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			return "1", fmt.Errorf("error scanning row: %w", err)
		}

		fmt.Printf("Fetched row: ID=%s, balance=%.2f, adminSum=%.2f, \n",
			id, balance.Float64, adminSum.Float64)
		rowCount++

		// Debug: Print the fetched row

		rowCount++

		// Update user_data Credit
		err = ExecuteQuery(ctx, "UPDATE user_data SET Credit = Credit - ? WHERE ID = ?", balance, id)
		if err != nil {
			return "", fmt.Errorf("error updating user_data for ID %s: %w", id, err)
		}
		fmt.Println("41PAST")
		// Update playinglog
		updateQuery := `UPDATE playinglog 
                        SET balance = ?, Game_Play = ?, Status = ?, adminSum = ?
                         WHERE ID = ? AND round = ? AND Game_play != ? AND Game_play != ?`
		err = ExecuteQuery(ctx, updateQuery, 0, "RENEW", "", 0, id, round-1, "END", "AFTER")
		if err != nil {
			return "2", fmt.Errorf("error updating playinglog for ID %s: %w", id, err)
		}
	}

	// Debug: Check row count
	if rowCount == 0 {
		fmt.Println("No rows fetched from playinglog.")
	}
	fmt.Println("4PAST")
	// Handle errors from rows iteration
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating over rows: %w", err)
	}
	query3 := `SELECT sub
			FROM playinglog
			WHERE 1
			ORDER BY Time DESC
			LIMIT 1`

	fmt.Printf("Executing query: %s\n", query)

	rows3, err := db.QueryContext(ctx, query3)
	if err != nil {
		log.Println("Error executing query: %v", err)
	}
	defer rows.Close()

	var sub3 int
	if rows3.Next() {
		if err3 := rows3.Scan(&sub3); err != nil {
			log.Println("Error scanning result: %v", err3)
		}
		fmt.Println("Sub:", sub3)
	} else {
		fmt.Println("No rows found")
	}

	// Update environment table FIXXX
	updateQuery := `
UPDATE environment 
SET 
    local_round = local_round - 1,
    local_state = 2, local_sub = ?
WHERE 1`

	// Make sure you're passing the correct value (sub3 + 1) to the query
	err4 := ExecuteQuery(ctx, updateQuery, sub3+1)
	if err4 != nil {
		return "", fmt.Errorf("error updating environment table: %w", err)
	}

	fmt.Println("5PAST")
	fmt.Println("Environment table updated successfully.") // Debug message
	return fmt.Sprintf("ประกาศผล คู่ที่ %d ใหม่ได้เลย", round-1), nil
}

func InsertUser(id, name, pic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Starting insertion process...")

	// ตรวจสอบจาก ID ก่อน (ไม่เช็ค name แล้ว)
	var existingID string
	var existingNumber string
	queryCheck := `
		SELECT id, Number
		FROM user_data
		WHERE id = ?
		LIMIT 1
	`
	err := db.QueryRowContext(ctx, queryCheck, id).Scan(&existingID, &existingNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			// ID ไม่ซ้ำ → แทรกผู้ใช้ใหม่
			queryMax := `
				SELECT COALESCE(MAX(Number), 0) + 1 AS newNumber
				FROM user_data
			`
			var newNumber int
			err := db.QueryRowContext(ctx, queryMax).Scan(&newNumber)
			if err != nil {
				log.Printf("Error fetching highest number: %v\n", err)
				return err
			}

			// ตรวจสอบว่าชื่อซ้ำกับผู้ใช้คนอื่นหรือไม่ (name ซ้ำแต่ ID ไม่ซ้ำ → ยอมให้แทรก)
			queryNameCheck := `
				SELECT COUNT(*) FROM user_data WHERE name = ?
			`
			var nameCount int
			err = db.QueryRowContext(ctx, queryNameCheck, name).Scan(&nameCount)
			if err != nil {
				log.Printf("Error checking for duplicate name: %v\n", err)
				return err
			}

			fmt.Printf("Name duplicate count: %d\n", nameCount)

			// แทรกได้ไม่ว่าชื่อจะซ้ำหรือไม่ (ชื่อซ้ำแต่ ID ไม่ซ้ำ = ยอมให้แทรก)
			queryInsert := `
				INSERT INTO user_data (id, name, user_profile, Number, Credit, Credit2)
				VALUES (?, ?, ?, ?, ?, ?)
			`
			fmt.Printf("Inserting new user: %s with Number: %d\n", name, newNumber)
			err = ExecuteQuery(ctx, queryInsert, id, name, pic, newNumber, "0", "0")
			if err != nil {
				log.Printf("Error inserting new user: %v\n", err)
				return err
			}
			fmt.Printf("Insertion successful for user: %s\n", name)

		} else {
			// Error อื่น ๆ
			log.Printf("Error checking for existing ID: %v\n", err)
			return err
		}
	} else {
		// ID ตรง → อัปเดต name และรูปโปรไฟล์
		queryUpdate := `
			UPDATE user_data
			SET name = ?, user_profile = ?
			WHERE id = ?
		`
		fmt.Printf("Updating existing user with ID: %s\n", id)
		err := ExecuteQuery(ctx, queryUpdate, name, pic, id)
		if err != nil {
			log.Printf("Error updating user with ID %s: %v\n", id, err)
			return err
		}
		fmt.Printf("Update successful for ID: %s\n", id)
	}

	return nil
}
func GetUserData(id string) (int64, int64, int64, error) {
	InitDB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error starting transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	query := `
        SELECT credit, credit2, Number
        FROM user_data
        WHERE id = ?
    `
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, fmt.Errorf("error preparing query: %v", err)
	}
	defer stmt.Close()

	// Use float64 for scanning
	var creditF, credit2F float64
	var numF sql.NullFloat64

	row := stmt.QueryRowContext(ctx, id)
	if err := row.Scan(&creditF, &credit2F, &numF); err != nil {
		return 0, 0, 0, fmt.Errorf("error retrieving user data: %v", err)
	}

	// Convert to int64
	credit := int64(creditF)
	credit2 := int64(credit2F)
	var num int64
	if numF.Valid {
		num = int64(numF.Float64)
	}

	fmt.Printf("Data retrieved successfully for user ID %s: credit=%d, credit2=%d, num=%d\n", id, credit, credit2, num)
	return credit, credit2, num, nil
}

func GetLocalVar() (int, int, int, float64, float64, bool, bool, string, int, int, int, error) {
	InitDB() // Ensure the database connection is initialized
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// SQL query to fetch the local variables, limited to 1 result
	query := `
        SELECT local_round, local_sub, local_state, local_red_rate, local_blue_rate,
               local_red_open, local_blue_open, local_command, local_min, local_max,local_win
        FROM environment
        LIMIT 1
    `

	// Execute the query
	row := db.QueryRowContext(ctx, query)

	// Declare variables to hold the data
	var localRound, localSub, localState int
	var localRedRateStr, localBlueRateStr string // Strings to hold fraction data
	var localRedOpen, localBlueOpen bool
	var localCommand string
	var localMin, localMax, local_win int

	// Scan the query result into variables
	err := row.Scan(&localRound, &localSub, &localState, &localRedRateStr, &localBlueRateStr,
		&localRedOpen, &localBlueOpen, &localCommand, &localMin, &localMax, &local_win)
	if err != nil {
		return 0, 0, 0, 0, 0, false, false, "", 0, 0, 0, fmt.Errorf("error retrieving data: %v", err)
	}

	// Parse the fraction strings into float64 values
	localRedRate, err := ParseFraction(localRedRateStr) // Parse fraction for red rate
	if err != nil {
		return 0, 0, 0, 0, 0, false, false, "", 0, 0, 0, fmt.Errorf("error parsing red rate: %v", err)
	}
	localBlueRate, err := ParseFraction(localBlueRateStr) // Parse fraction for blue rate
	if err != nil {
		return 0, 0, 0, 0, 0, false, false, "", 0, 0, 0, fmt.Errorf("error parsing blue rate: %v", err)
	}

	// Return the data
	return localRound, localSub, localState, localRedRate, localBlueRate, localRedOpen,
		localBlueOpen, localCommand, localMin, localMax, local_win, nil
}

func ParseFraction(fraction string) (float64, error) {
	parts := strings.Split(fraction, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid fraction format")
	}
	numerator, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid numerator: %v", err)
	}
	denominator, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid denominator: %v", err)
	}
	if denominator == 0 {
		return 0, fmt.Errorf("denominator cannot be zero")
	}
	return float64(numerator) / float64(denominator), nil
}
func convertToInt(value interface{}) (int, error) {
	strValue, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("expected string, got %T", value)
	}
	return strconv.Atoi(strValue)
}
func getMessage(localWin int, localRedRate, localBlueRate, localMax string) string {
	var message string

	if localWin <= 0 {
		message = fmt.Sprintf(
			"🔴เล่นแดง%sราคา       %s\n🔵เล่นน้ำเงิน%sราคา    %s\n รับ %s",
			"ต่อ", localRedRate, "รอง", localBlueRate, localMax,
		)
	} else {
		message = fmt.Sprintf(
			"🔵เล่นน้ำเงิน%sราคา    %s\n🔴เล่นแดง%sราคา       %s\n รับ %s",
			"ต่อ", localBlueRate, "รอง", localRedRate, localMax,
		)
	}

	return message
}
func getMessage2(localWin int, localRedRate, localBlueRate, localMax string) string {
	var message string

	if localWin <= 0 {
		message = fmt.Sprintf(
			"🔴เล่นแดง%sราคา       %s\n🔵เล่นน้ำเงิน%sราคา    %s\n รับ %s",
			"ต่อ", localRedRate, "ต่อ", localBlueRate, localMax,
		)
	} else {
		message = fmt.Sprintf(
			"🔵เล่นน้ำเงิน%sราคา    %s\n🔴เล่นแดง%sราคา       %s\n รับ %s",
			"ต่อ", localBlueRate, "ต่อ", localRedRate, localMax,
		)
	}

	return message
}
func getMessage3(localWin int, localRedRate, localBlueRate, localMax string) string {
	var message string

	if localWin == 1 {
		message = fmt.Sprintf("...\n🔴รองแดงราคา %s\nรับ %s", localRedRate, localMax)
	} else {
		message = fmt.Sprintf("...\n🔵รองน้ำเงินราคา %s\nรับ %s", localBlueRate, localMax)
	}

	return message
}

// CONDITION CHECK
func ifThenElse(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}
func IntPtr(i int) *int {
	return &i
}
func formatWithCommasX(input string) string {
	n, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		fmt.Println("Error parsing number:", err)
		return input
	}
	return strconv.FormatInt(n, 10) // Returns formatted string with commas
}
func truncate(s string, maxLength int) string {
	if utf8.RuneCountInString(s) <= maxLength {
		return s
	}

	runes := []rune(s)
	return string(runes[:maxLength])
}
func findGCD(a, b int) int {
	if a == b {
		return 1 // บังคับให้ผลลัพธ์เป็น 1 ถ้า a == b
	}
	if b == 0 {
		return a
	}
	return findGCD(b, a%b)
}

// Function to convert a float into a simplified fraction
func floatToFraction(f float64) (int, int) {
	// Handle special case: 1.0 -> return 10/10
	if math.Abs(f-1.0) <= 0.0001 {
		return 10, 10
	}
	log.Println("OKK", math.Abs(f-1.0))
	// Scale the number to handle up to 4 decimal places (precision)
	precision := 10000 // 4 decimal places
	scaled := int(f * float64(precision))

	// Handle specific cases for repeating decimals
	if math.Abs(f-1.6666666666666667) < 0.0001 {
		return 5, 3
	}
	if math.Abs(f-1.1111111111111112) < 0.0001 {
		return 10, 9
	}
	if math.Abs(f-1.5) < 0.0001 {
		return 3, 2
	}
	if math.Abs(f-2.5) < 0.0001 {
		return 5, 2
	}

	// General conversion to fraction
	numerator := scaled
	denominator := precision

	// Simplify the fraction using GCD
	gcd := findGCD(numerator, denominator)
	return numerator / gcd, denominator / gcd
}

func CheckMessage(rawMessage string, userID string, groupID string, replyToken string) string {

	displayName, pictureURL, statusMessage, _ := getUserProfile(groupID, userID)
	if displayName != "" {
		InsertUser(userID, displayName, pictureURL)
	}

	localRound, localSub, localState, localRedRate, localBlueRate, localRedOpen,
		localBlueOpen, localCommand, localMin, localMax, localWin, err := GetLocalVar()
	if err != nil {
		fmt.Println("Error:", err)
		return displayName
	}
	if groupID == groupPlay && (strings.ToLower(strings.Trim(rawMessage, " ")) == "c" || strings.ToLower(strings.Trim(rawMessage, " ")) == "cc" || strings.ToLower(rawMessage) == "u" || strings.ToLower(rawMessage) == "game" || strings.ToLower(rawMessage) == "หลังบ้าน") {
		return ""
	}
	if groupID == groupC && (strings.ToLower(rawMessage) == "u" || strings.ToLower(rawMessage) == "game" || strings.ToLower(rawMessage) == "หลังบ้าน") {
		return ""
	}

	fmt.Println("Display Name:", displayName)
	fmt.Println("Picture URL:", pictureURL)
	fmt.Println("Status Message:", statusMessage)
	cleanedMessage := strings.ReplaceAll(rawMessage, " ", "")
	cleanedMessage = strings.ReplaceAll(cleanedMessage, "\u200B", "") // Remove zero-width space
	cleanedMessage = strings.ReplaceAll(cleanedMessage, ",", "")      // Remove zero-width space
	cleanedMessage = strings.ReplaceAll(cleanedMessage, "/", "")      // Remove zero-width space
	cleanedMessage = strings.ReplaceAll(cleanedMessage, "\n", "")     // Remove zero-width space
	var adminPower, _ = CheckAdmin(userID)
	// Define the pattern to match a character (ด, ต, ง) followed by a number
	pattern := `^([ดตง])[\n/\\]?(\d+)$`
	re := regexp.MustCompile(pattern)

	// Check if the cleaned raw message matches the pattern
	if match := re.FindStringSubmatch(cleanedMessage); match != nil && re.MatchString(cleanedMessage) {
		// Extract the matched character (e.g., "ง") and the number (e.g., "100")

		// matchedText := match[1] // Captured character (e.g., "ง")
		// number := match[2]      // Captured number (e.g., "100")
		if localState == 2 {
			if showReply {
				return ""
			} else {
				return "ปิดรับการเดิมพันแล้ว"
			}
			return "ปิดรับการเดิมพันแล้ว"

		}
		if localState == 1 {
			matchedText := match[1]
			if localBlueOpen && matchedText == "ง" {
				return bet(userID, cleanedMessage, getBalance(userID), displayName)
			}
			if localRedOpen && matchedText == "ด" {
				return bet(userID, cleanedMessage, getBalance(userID), displayName)

			}
			return fmt.Sprintf("❌ไม่ติด❌\nไม่ได้เปิดรับมุมตรงข้าม")
		} else {
			log.Println("ERROR APPEARED!!")
		}

		return ""
	}

	// Define regular expressions
	if adminPower == 1 {

		// fmt.Println("AUOK", rawMessage[0])

		credit, credit2, num64, err := GetUserData(userID)
		// fmt.Println("AUOK", num64)
		num := int(num64)
		runes := []rune(rawMessage)
		if len(rawMessage) > 3 {
			if len(rawMessage) > 3 && rawMessage[0] == '@' && !strings.Contains(rawMessage, "=") && !strings.Contains(rawMessage, "ยกแผล") && !strings.Contains(rawMessage, "เช็ค") && !strings.Contains(rawMessage, "+") {
				ctx := context.Background()
				// Get user profile
				var profile *linebot.UserProfileResponse
				if groupID != "" {
					profile, err = lineBotAPI.GetGroupMemberProfile(groupID, userID).Do()
				} else {
					profile, err = lineBotAPI.GetProfile(userID).Do()
				}
				if err != nil {
					log.Println(err)
					return ""
				}

				// Regular expression for parsing the message
				pattern := `@([^/]+)\s*/\s*(-?\d+(?:,\d{3})*(?:\.\d+)?)$`
				re := regexp.MustCompile(pattern)

				// Check for a match
				match := re.FindStringSubmatch(rawMessage)
				if match == nil {
					return ""
				}

				username := strings.TrimSpace(match[1])             // Extract the username
				amountStr := strings.Replace(match[2], ",", "", -1) // Remove commas from the amount
				amount, err := strconv.ParseFloat(amountStr, 64)    // Convert to float64
				if err != nil {
					log.Printf("Error parsing amount: %v", err)
					return ""
				}

				// Update user data
				query := `
					UPDATE user_data 
SET Credit = Credit + ? 
WHERE TRIM(Name) = TRIM(?) 
AND (
    SELECT COUNT(*) 
    FROM user_data 
    WHERE TRIM(Name) = TRIM(?)
) = 1
`
				err = ExecuteQuery(ctx, query, amount, username, username)
				if err != nil {
					log.Printf("Error updating user data: %v", err)
					return ""
				}

				// Check if any row was updated
				var rowCount int
				err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_data WHERE TRIM(Name) = TRIM(?)`, username).Scan(&rowCount)
				if err != nil {
					log.Printf("Error checking row count: %v", err)
					return ""
				}

				if rowCount == 0 {
					flexMessage := &linebot.FlexMessage{
						AltText: "Name doesn't exist or it's a duplicate",
						Contents: &linebot.BubbleContainer{
							Body: &linebot.BoxComponent{
								Type:   "box",
								Layout: "vertical",
								Contents: []linebot.FlexComponent{
									&linebot.TextComponent{
										Type:   "text",
										Text:   "ชื่อในระบบซ้ำกัน กรุณาใช้ ID ในการฝาก/ถอน",
										Weight: "bold",
										Size:   "lg",
										Color:  "#FF0000",
									},
								},
							},
						},
					}
					_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
					if err != nil {
						log.Printf("Error sending reply message: %v", err)
					}
					return ""
				}

				// Insert into admin log
				adminLogQuery := `INSERT INTO adminlog (ID, action) VALUES (?, ?)`
				err = ExecuteQuery(ctx, adminLogQuery, profile.UserID, fmt.Sprintf("Line ฝาก %s", amountStr))
				if err != nil {
					log.Printf("Error inserting admin log: %v", err)
					return ""
				}

				// Insert withdrawal log
				state := "ฝาก"
				if amountStr[0] == '-' {
					state = "ถอน"
				}
				var idNum int // Declare the variable to store the result

				// Execute the query
				err4 := db.QueryRowContext(ctx, `SELECT number FROM user_data WHERE Name = ?`, username).Scan(&idNum)
				if err4 != nil {
					if err4 == sql.ErrNoRows {
						// Handle case where no rows are returned
						log.Printf("No user found with Name: %s", username)
						return "0" // You might return an error or 0 depending on your use case
					}
					// Log and return other errors
					log.Printf("Error fetching user number: %v", err)
					return "0"
				}

				wdLogQuery := `INSERT INTO wd (UID, STATE, NAME, AMOUNT, note) VALUES (?, ?, ?, ?, ?)`
				err = ExecuteQuery(ctx, wdLogQuery, idNum, state, username, amountStr, "LINE")
				if err != nil {
					log.Printf("Error inserting withdrawal log: %v", err)
					return ""
				}
				actionP := linebot.NewURIAction("ไปห้องเล่น", PlayRoom)
				actionD := linebot.NewURIAction("ไปห้องฝาก", DepositRoom)
				// Fetch updated credit
				var credit float64
				err = db.QueryRowContext(ctx, `SELECT Credit FROM user_data WHERE Name = ?`, username).Scan(&credit)
				if err != nil {
					log.Printf("Error fetching updated credit: %v", err)
					return ""
				}

				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					"จัดการสำเร็จ",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:            "box",
							Layout:          "vertical",
							Spacing:         "sm",
							PaddingAll:      "md",
							BackgroundColor: "#000000", // พื้นหลังดำ
							BorderWidth:     "light",
							BorderColor:     "#ffc60a",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "จัดการยอดฝาก",
									Weight: "bold",
									Size:   "md",
									Color:  "#FFD700", // ทอง
									Margin: "xs",
									Align:  "center",
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้ใช้:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  username,
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "จำนวน:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(fmt.Sprintf("%.0f", amount)) + "฿",
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "คงเหลือ:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)) + "฿",
											Size:  "xs",
											Color: "#00FF00", // เขียวสด
											Align: "end",
										},
									},
								},
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#FFD700", // เส้นทอง
									Margin: "sm",
								},
								&linebot.BoxComponent{
									Type:        "box",
									Layout:      "horizontal",
									Spacing:     "sm",
									BorderWidth: "none",
									Contents: []linebot.FlexComponent{
										&linebot.BoxComponent{
											Type:            "box",
											Layout:          "vertical",
											CornerRadius:    "sm",
											BorderWidth:     "light",
											BorderColor:     "#ffc60a",
											AlignItems:      "center",
											BackgroundColor: "#ff0000",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type:  "text",
													Text:  "ไปห้องเล่น",
													Size:  "xxs",
													Color: "#ffffff",
													Wrap:  true,
												},
											},
											Action: actionP,
										},
										&linebot.BoxComponent{
											Type:            "box",
											Layout:          "vertical",
											CornerRadius:    "sm",
											BorderWidth:     "light",
											BorderColor:     "#ffc60a",
											AlignItems:      "center",
											BackgroundColor: "#00c000",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type:  "text",
													Text:  "ไปห้องฝาก",
													Size:  "xxs",
													Color: "#ffffff",
													Wrap:  true,
												},
											},
											Action: actionD,
										},
									},
								},
							},
						},
						Styles: &linebot.BubbleStyle{
							Body: &linebot.BlockStyle{
								BackgroundColor: "#000000", // พื้นหลังดำ
							},
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

				return "success"
			} else if rawMessage[0] == '@' && strings.Contains(rawMessage, "=") {
				ctx := context.Background()
				// Get user profile
				var profile *linebot.UserProfileResponse
				if groupID != "" {
					profile, err = lineBotAPI.GetGroupMemberProfile(groupID, userID).Do()
				} else {
					profile, err = lineBotAPI.GetProfile(userID).Do()
				}

				if err != nil {
					log.Println(err)
					return ""
				}

				// Regular expression for parsing the message
				pattern := `@([^/]+)\s*[\s/=]\s*(-?\d+(?:,\d{3})*(?:\.\d+)?)$`
				re := regexp.MustCompile(pattern)

				// Check for a match
				match := re.FindStringSubmatch(rawMessage)
				if match == nil {
					return ""
				}

				username := strings.TrimSpace(match[1])             // Extract the username
				amountStr := strings.Replace(match[2], ",", "", -1) // Remove commas from the amount

				// Convert amount to a float to handle decimals and negative values
				amount, err := strconv.ParseFloat(amountStr, 64)
				if err != nil {
					log.Printf("Error parsing amount: %v", err)
					return ""
				}

				// Update user data
				query := `
					UPDATE user_data 
					SET Credit = Credit + ?, Credit2 = Credit2 + ? 
					WHERE TRIM(Name) = TRIM(?) 
					AND (SELECT COUNT(*) FROM user_data WHERE TRIM(Name) =TRIM(?) ) = 1`
				err = ExecuteQuery(ctx, query, amount, amount, username, username)
				if err != nil {
					log.Printf("Error updating user data for %s: %v", username, err)
					return ""
				}

				// Check if any row was updated
				var rowCount int
				err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_data WHERE TRIM(Name) = TRIM(?) `, username).Scan(&rowCount)
				if err != nil {
					log.Printf("Error checking row count for %s: %v", username, err)
					return ""
				}

				if rowCount == 0 {
					flexMessage := &linebot.FlexMessage{
						AltText: "Name doesn't exist or it's a duplicate",
						Contents: &linebot.BubbleContainer{
							Size: "micro",
							Body: &linebot.BoxComponent{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: "#000000", // พื้นหลังดำ
								Contents: []linebot.FlexComponent{
									&linebot.TextComponent{
										Type:   "text",
										Text:   "ชื่อในระบบซ้ำกัน\nใช้คำสั่งผ่าน ID ในการดำเนินการแทน",
										Weight: "bold",
										Size:   "md",
										Color:  "#FFD700", // สีทอง
										Wrap:   true,
										Align:  "center",
									},
								},
							},
							Styles: &linebot.BubbleStyle{
								Body: &linebot.BlockStyle{
									BackgroundColor: "#000000",
								},
							},
						},
					}
					_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
					if err != nil {
						log.Printf("Error sending reply message1: %v", err)
					}
					return ""
				}

				// Insert into admin log
				adminLogQuery := `INSERT INTO adminlog (ID, action) VALUES (?, ?)`
				err = ExecuteQuery(ctx, adminLogQuery, profile.UserID, fmt.Sprintf("Line ฝาก %.2f", amount))
				if err != nil {
					log.Printf("Error inserting admin log: %v", err)
					return ""
				}

				// Insert withdrawal log
				state := "เติมเครดิต"
				if amount < 0 {
					state = "ถอนเครดิต"
				}
				wdLogQuery := `INSERT INTO wd (UID, STATE, NAME, AMOUNT, note) VALUES (?, ?, ?, ?, ?)`
				err = ExecuteQuery(ctx, wdLogQuery, profile.UserID, state, username, amount, "LINE")
				if err != nil {
					log.Printf("Error inserting withdrawal log: %v", err)
					return ""
				}

				// Fetch updated credit
				var credit float64
				err = db.QueryRowContext(ctx, `SELECT Credit2 FROM user_data WHERE Name = ?`, username).Scan(&credit)
				if err != nil {
					log.Printf("Error fetching updated credit/2 for %s: %v", username, err)
					return ""
				}
				actionP := linebot.NewURIAction("ไปห้องเล่น", PlayRoom)
				actionD := linebot.NewURIAction("ไปห้องฝาก", DepositRoom)
				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					"จัดการเครดิต",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:            "box",
							Layout:          "vertical",
							Spacing:         "sm",
							PaddingAll:      "md",
							BackgroundColor: "#000000", // พื้นหลังดำ
							BorderWidth:     "light",
							BorderColor:     "#ffc60a",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "จัดการเครดิต",
									Weight: "bold",
									Size:   "md",
									Color:  "#FFD700", // ทอง
									Margin: "xs",
									Align:  "center",
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้ใช้:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  username,
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "จำนวน:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(fmt.Sprintf("%.0f", amount)) + "฿",
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:    "box",
									Layout:  "horizontal",
									Spacing: "sm",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "เครดิตคงเหลือ:",
											Size:  "xs",
											Color: "#FFD700",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)) + "฿",
											Size:  "xs",
											Color: "#00FF00", // เขียวสด
											Align: "end",
										},
									},
								},
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#FFD700", // เส้นทอง
									Margin: "sm",
								},
								&linebot.BoxComponent{
									Type:        "box",
									Layout:      "horizontal",
									Spacing:     "sm",
									BorderWidth: "none",
									Contents: []linebot.FlexComponent{
										&linebot.BoxComponent{
											Type:            "box",
											Layout:          "vertical",
											CornerRadius:    "sm",
											BorderWidth:     "light",
											BorderColor:     "#ffc60a",
											AlignItems:      "center",
											BackgroundColor: "#ff0000",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type:  "text",
													Text:  "ไปห้องเล่น",
													Size:  "xxs",
													Color: "#ffffff",
													Wrap:  true,
												},
											},
											Action: actionP,
										},
										&linebot.BoxComponent{
											Type:            "box",
											Layout:          "vertical",
											CornerRadius:    "sm",
											BorderWidth:     "light",
											BorderColor:     "#ffc60a",
											AlignItems:      "center",
											BackgroundColor: "#00c000",
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type:  "text",
													Text:  "ไปห้องฝาก",
													Size:  "xxs",
													Color: "#ffffff",
													Wrap:  true,
												},
											},
											Action: actionD,
										},
									},
								},
							},
						},
						Styles: &linebot.BubbleStyle{
							Body: &linebot.BlockStyle{
								BackgroundColor: "#000000", // พื้นหลังดำ
							},
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

				return "success"
			} else if rawMessage[0] == '@' && strings.Contains(rawMessage, "ยกแผล") {
				ctx := context.Background()
				// Get user profile
				// profile, err := lineBotAPI.GetGroupMemberProfile(groupID, userID).Do()

				// Regular expression for parsing the message
				pattern := `@([^/]+)\s*ยกแผล`
				re := regexp.MustCompile(pattern)
				fmt.Println("CANCEL is in PROGRESS")
				// Check for a match
				match := re.FindStringSubmatch(rawMessage)
				if match == nil {
					return ""
				}

				username := strings.TrimSpace(match[1]) // Extract the username

				if err != nil {
					log.Printf("Error parsing amount: %v", err)
					return ""
				}

				// Update user data
				query := `
					SELECT b1, j1, red_rate, blue_rate
					FROM playinglog
					WHERE TRIM(Name) = TRIM(?)  AND balance = '0'
					ORDER BY Time DESC
					LIMIT 1
				`

				// Fetch the row
				var b1, j1 float64
				var redRate, blueRate string
				row := db.QueryRowContext(ctx, query, username)
				err6 := row.Scan(&b1, &j1, &redRate, &blueRate)
				if err6 != nil {
					if err6 == sql.ErrNoRows {
						// No rows found for the specified username
						fmt.Println("No rows found for the specified username")

						// Assign default values
						b1, j1, redRate, blueRate = 0, 0, "1", "1"
					} else {
						// Handle other types of errors
						log.Println("Error fetching row: %v\n", err6)
					}
				}

				// Successfully fetched or assigned default values

				fmt.Printf("Fetched row: b1=%f, j1=%f, red_rate=%s, blue_rate=%s\n", b1, j1, redRate, blueRate)

				var sideHead = "ด"

				if j1 > b1 {
					b1 = j1
					sideHead = "ง"
					redRate = blueRate
				}

				redRate2, _ := strconv.ParseFloat(redRate, 64)
				demo := 1 // Example value

				// Convert redRate2 into a simplified fraction
				numerator, denominator := floatToFraction(redRate2)

				// Adjust demo proportionally
				demoNumerator := numerator * demo
				demoDenominator := denominator

				// Simplify the resulting fraction
				gcd := findGCD(demoNumerator, demoDenominator)
				demoNumerator /= gcd
				demoDenominator /= gcd
				var amount = sideHead + formatWithCommas2(int(b1)) + fmt.Sprintf(" (%v/%v)", demoNumerator, demoDenominator)
				if j1 == b1 && j1 == 0 {
					amount = "ไม่มีแผลในระบบแล้ว"
				}
				// Delete the row
				deleteQuery := `
					DELETE FROM playinglog
					WHERE Name = ? AND balance = '0'
					ORDER BY Time DESC
					LIMIT 1
				`
				result, err := db.ExecContext(ctx, deleteQuery, username)
				if err != nil {
					log.Println("Error deleting row: %v\n", err)
				}

				// Confirm deletion
				rowsAffected, err := result.RowsAffected()
				if err != nil {
					log.Println("Error fetching rows affected: %v\n", err)
				}

				if rowsAffected > 0 {
					fmt.Printf("Row successfully deleted. Rows affected: %d\n", rowsAffected)
				} else {
					fmt.Println("No rows were deleted (possibly already removed).")
				}

				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					"ยกเลิกแผลล่าสุดสำเร็จ",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "จัดการยกเลิกการเดิมพัน",
									Weight: "bold",
									Size:   "md",
									Color:  "#FFD700", // สีทอง
									Margin: "xs",
									Align:  "center",
								},
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#FFD700", // ขีดทองคั่น
									Margin: "sm",
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:   "text",
											Text:   "ผู้ใช้:",
											Size:   "xs",
											Color:  "#FFD700",
											Weight: "bold",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  username,
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:   "text",
											Text:   "แผลที่ยกเลิก:",
											Size:   "xs",
											Color:  "#FFD700",
											Weight: "bold",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  amount,
											Size:  "xs",
											Color: "#FFFFFF",
											Align: "end",
										},
									},
								},
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#FFD700",
									Margin: "sm",
								},
								&linebot.TextComponent{
									Type:   "text",
									Text:   "ดำเนินการแล้ว",
									Weight: "bold",
									Size:   "md",
									Color:  "#FFD700",
									Margin: "md",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#000000", // พื้นหลังดำสนิท
							BorderColor:     "#FFD700", // กรอบทอง
							BorderWidth:     "medium",
							CornerRadius:    "lg",
							PaddingAll:      "md",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

				return "success"
			} else if rawMessage[0] == '@' && strings.Contains(rawMessage, "เช็ค") {
				ctx := context.Background()
				// Get user profile
				// profile, err := lineBotAPI.GetGroupMemberProfile(groupID, userID).Do()

				// Regular expression for parsing the message
				fmt.Println("Data is in PROGRESS")
				pattern := `@([^/]+)\s*เช็ค\s*(\d{1,4})([-.]?)`
				re := regexp.MustCompile(pattern)

				// Check for a match
				match := re.FindStringSubmatch(rawMessage)
				if match == nil {
					return ""
				}

				username := (match[1])                     // Extract the username
				wantedRound := strings.TrimSpace(match[2]) // Extract the round
				if err != nil {
					log.Printf("Error parsing amount: %v", err)
					return ""
				}
				// Update user data
				// Updated query to fetch the `balance` column
				query := `
    SELECT b1, j1, red_rate, blue_rate, balance,Time2,win 
    FROM playinglog
    WHERE TRIM(Name) = TRIM(?)  AND round =?  AND Game_play != "END"
    ORDER BY Time2 DESC
`
				// Declare the variables for row data
				var b1, j1 float64
				var redRate, blueRate, wantedTime string
				var balance int
				var wantedWin int
				var wantedRate string

				// Execute the query
				rows, err := db.QueryContext(ctx, query, username, wantedRound)
				if err != nil {
					log.Println("Error executing query: %v\n", err)
				}
				defer rows.Close()

				// Initialize values
				var sideHead = "ด"
				// var balanceStatus string
				var amount string
				found := false

				// Loop through all rows returned by the query
				amount = ""
				for rows.Next() {
					err := rows.Scan(&b1, &j1, &redRate, &blueRate, &balance, &wantedTime, &wantedWin)
					if err != nil {
						log.Println("Error scanning row: %v\n", err)
					}

					// Update balance status
					// if balance > 0 {
					// 	balanceStatus = "ได้"
					// } else {
					// 	balanceStatus = "เสีย"
					// }
					sideHead = "ด"
					// Update the sideHead and redRate if needed
					if j1 > b1 {
						b1 = j1
						sideHead = "ง"
						redRate = blueRate
					}
					if sideHead == "ด" {
						if wantedWin == 1 {
							wantedRate = "รอง"
						} else {
							wantedRate = "ต่อ"
						}
					} else {
						if wantedWin == -1 {
							wantedRate = "รอง"
						} else {
							wantedRate = "ต่อ"
						}
					}
					t, err := time.Parse("2006-01-02 15:04:05.999", wantedTime)
					// var demo int

					// demo = 1
					wantedTime = t.Format("15:04")
					// redRate := 1.75 // Example value
					redRate2, _ := strconv.ParseFloat(redRate, 64)
					demo := 1 // Example value

					// Convert redRate2 into a simplified fraction
					numerator, denominator := floatToFraction(redRate2)

					// Adjust demo proportionally
					demoNumerator := numerator * demo
					demoDenominator := denominator

					// Simplify the resulting fraction
					gcd := findGCD(demoNumerator, demoDenominator)
					demoNumerator /= gcd
					demoDenominator /= gcd

					// Construct the amount string for this row
					amount += sideHead + formatWithCommas2(int(b1)) +
						fmt.Sprintf(" (%v %d/%d)  เมื่อ: (%s) \n", wantedRate, demoNumerator, demoDenominator, wantedTime)

					// Check for empty results in the system
					if j1 == b1 && j1 == 0 {
						amount = "ไม่มีแผลในระบบ"
					}

					// Print results
					fmt.Printf("Fetched row: b1=%f, j1=%f, red_rate=%d/%d, blue_rate=%s, balance=%d\n",
						b1, j1, demoNumerator, demoDenominator, blueRate, balance)
					fmt.Printf("Amount: %s\n", amount)

					found = true

				}

				// Check if no rows were found
				if !found {
					fmt.Println("No rows found for the specified username")
					amount = "ไม่มีแผลในระบบ"
				} else {
					amount += fmt.Sprintf(" รวม : %v ฿ ", balance)
				}

				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					"✅ ข้อมูลคู่การแทง ✅",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ข้อมูลคู่การแทง ✅",
									Weight: "bold",
									Size:   "md",
									Color:  "#1DB446",
									Margin: "xs",
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้ใช้:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  username,
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										// &linebot.TextComponent{
										// 	Type:  "text",
										// 	Text:  "แผล:",
										// 	Size:  "xs",
										// 	Color: "#ffffff",
										// },
										&linebot.TextComponent{
											Type:  "text",
											Text:  amount,
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",

											Wrap: true,
										},
									},
								},
								// &linebot.BoxComponent{
								// 	Type:   "box",
								// 	Layout: "horizontal",
								// 	Contents: []linebot.FlexComponent{
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  "เครดิตรวม:",
								// 			Size:  "xs",
								// 			Color: "#ffffff",
								// 		},
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)),
								// 			Size:  "xs",
								// 			Color: "#1DB446",
								// 			Align: "end",
								// 		},
								// 	},
								// },
								// &linebot.SeparatorComponent{
								// 	Type:   "separator",
								// 	Color:  "#ffffff", // White line
								// 	Margin: "sm",
								// },
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ทำรายการสำเร็จ",
									Weight: "bold",
									Size:   "md",
									Color:  "#1DB446",
									Margin: "xs",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#222222",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

				return "success"
			} else if rawMessage[0] == '$' && strings.Contains(rawMessage, "เช็ค") {
				ctx := context.Background()
				fmt.Println("Data is in PROGRESS")

				// Regular expression for parsing the message
				pattern := `\$(\d+)\s*เช็ค\s*(\d{1,4})([-.]?)`
				re := regexp.MustCompile(pattern)

				// Check for a match
				match := re.FindStringSubmatch(rawMessage)
				if match == nil {
					return ""
				}

				userNumber := strings.TrimSpace(match[1])  // Extract the user number
				wantedRound := strings.TrimSpace(match[2]) // Extract the round
				var userName string

				// First, get the user ID from user_table based on the number
				var userID string
				err := db.QueryRowContext(ctx, "SELECT ID,Name FROM user_data WHERE Number = ?", userNumber).Scan(&userID, &userName)
				if err != nil {
					if err == sql.ErrNoRows {
						log.Printf("User with number %s not found", userNumber)
						return "ไม่พบผู้ใช้หมายเลข " + userNumber
					}
					log.Printf("Error querying user ID: %v", err)
					return ""
				}

				// Query to fetch the playing log using the user ID
				query := `
					SELECT b1, j1, red_rate, blue_rate, balance, Time2, win 
					FROM playinglog
					WHERE ID = ? AND round = ? AND Game_play != "END"
					ORDER BY Time2 DESC
				`

				// Declare the variables for row data
				var b1, j1 float64
				var redRate, blueRate, wantedTime string
				var balance int
				var wantedWin int
				var wantedRate string

				// Execute the query
				rows, err := db.QueryContext(ctx, query, userID, wantedRound)
				if err != nil {
					log.Printf("Error executing query: %v\n", err)
				}
				defer rows.Close()

				// Initialize values
				var sideHead = "ด"
				var amount string
				found := false

				amount = ""
				for rows.Next() {
					err := rows.Scan(&b1, &j1, &redRate, &blueRate, &balance, &wantedTime, &wantedWin)
					if err != nil {
						log.Printf("Error scanning row: %v\n", err)
					}

					sideHead = "ด"
					if j1 > b1 {
						b1 = j1
						sideHead = "ง"
						redRate = blueRate
					}

					if sideHead == "ด" {
						if wantedWin == 1 {
							wantedRate = "รอง"
						} else {
							wantedRate = "ต่อ"
						}
					} else {
						if wantedWin == -1 {
							wantedRate = "รอง"
						} else {
							wantedRate = "ต่อ"
						}
					}

					t, err := time.Parse("2006-01-02 15:04:05.999", wantedTime)
					if err != nil {
						log.Printf("Error parsing time: %v", err)
					}
					wantedTime = t.Format("15:04")

					redRate2, _ := strconv.ParseFloat(redRate, 64)
					demo := 1

					numerator, denominator := floatToFraction(redRate2)
					demoNumerator := numerator * demo
					demoDenominator := denominator

					gcd := findGCD(demoNumerator, demoDenominator)
					demoNumerator /= gcd
					demoDenominator /= gcd

					amount += sideHead + formatWithCommas2(int(b1)) +
						fmt.Sprintf(" (%v %d/%d)  เมื่อ: (%s) \n", wantedRate, demoNumerator, demoDenominator, wantedTime)

					if j1 == b1 && j1 == 0 {
						amount = "ไม่มีแผลในระบบ"
					}

					fmt.Printf("Fetched row: b1=%f, j1=%f, red_rate=%d/%d, blue_rate=%s, balance=%d\n",
						b1, j1, demoNumerator, demoDenominator, blueRate, balance)
					fmt.Printf("Amount: %s\n", amount)

					found = true
				}

				if !found {
					fmt.Println("No rows found for the specified user ID")
					amount = "ไม่มีแผลในระบบ"
				} else {
					amount += fmt.Sprintf(" รวม : %v ฿ ", balance)
				}

				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					"✅ ข้อมูลคู่การแทง ✅",
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ข้อมูลคู่การแทง ✅",
									Weight: "bold",
									Size:   "md",
									Color:  "#1DB446",
									Margin: "xs",
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้ใช้:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  fmt.Sprintf("(%v)%v", userNumber, userName),
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  amount,
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
											Wrap:  true,
										},
									},
								},
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ทำรายการสำเร็จ",
									Weight: "bold",
									Size:   "md",
									Color:  "#1DB446",
									Margin: "xs",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#222222",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

				return "success"
			} else if len(runes) >= 4 && string(runes[:2]) == "ผล" {
				if localState == 1 || localState == 0 {
					return ""
				}
				if string(runes[3:]) == "ด" {
					te := Summarize2("R", round)
					// flexMessage, _ := GenerateFlexMessage(te)
					// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
					// // playerData, err := parsePlayerData(input)
					// if err != nil {
					// 	fmt.Println("Error parsing input:", err)
					// 	return input
					// }

					// Generate Flex message
					flexMessage, _ := GenerateFlexMessage2(te, localRound, "แดงชนะ")
					bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
					if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
						log.Println(err)
					}

					log.Println("Flex message sent successfully")
					te2 := fmt.Sprintf("%v", te)
					nextRound()

					return te2
				}
				if string(runes[3:]) == "ส" {
					te := Summarize2("S", round)
					// flexMessage, _ := GenerateFlexMessage(te)
					// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
					// // playerData, err := parsePlayerData(input)
					// if err != nil {
					// 	fmt.Println("Error parsing input:", err)
					// 	return input
					// }

					// Generate Flex message
					flexMessage, _ := GenerateFlexMessage2(te, localRound, "เสมอ")
					bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
					if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
						log.Println(err)
					}

					log.Println("Flex message sent successfully")
					te2 := fmt.Sprintf("%v", te)
					nextRound()
					return te2
				}
				if string(runes[3:]) == "ง" {
					te := Summarize2("B", round)
					// flexMessage, _ := GenerateFlexMessage(te)
					// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
					// // playerData, err := parsePlayerData(input)
					// if err != nil {
					// 	fmt.Println("Error parsing input:", err)
					// 	return input
					// }

					// Generate Flex message
					flexMessage, _ := GenerateFlexMessage2(te, localRound, "น้ำเงินชนะ")
					bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
					if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
						log.Println(err)
					}

					log.Println("Flex message sent successfully")
					te2 := fmt.Sprintf("%v", te)
					nextRound()
					return te2
				}
				return ""
			}

			if len(runes) >= 4 && string(runes[:1]) == "$" && (strings.Contains(rawMessage, "+") || strings.Contains(rawMessage, "-")) {
				// Process valid URLs or responses from the Python script
				re := regexp.MustCompile(`^\$(\d+)([+-])([\d,]+)$`)

				matches := re.FindStringSubmatch(rawMessage)
				// if len(matches) != 3 {
				// 	return ""
				// }

				// Extract account number
				accountNumber, err := strconv.Atoi(matches[1])
				if err != nil {
					return "ทำรายการไม่สำเร็จ ID ไม่ถูกรูปแบบ"
				}

				// Extract amount and remove commas

				amountStr := strings.ReplaceAll(matches[3], ",", "")
				// amount, err := strconv.ParseFloat(amountStr, 64)
				if matches[2] == "-" {
					amountStr2, _ := strconv.Atoi(amountStr)
					amountStr = fmt.Sprintf("-%v", amountStr2)
				}
				// if err != nil {
				// 	return "ชุดข้อมูลไม่ถูกต้อง"
				// }
				result := amountStr
				ctx := context.Background()
				fmt.Println("Valid response received:", result)
				if result == "0" {

				}
				var idNum, idName string // Declare the variable to store the result

				// Execute the query
				_ = db.QueryRowContext(ctx, `SELECT ID, Name FROM user_data WHERE Number = ?`, accountNumber).Scan(&idNum, &idName)

				query := `
						UPDATE user_data 
						SET Credit = Credit + ? 
						WHERE Number = ? 
						`
				err = ExecuteQuery(ctx, query, result, accountNumber)
				if err != nil {
					log.Printf("Error updating user data: %v", err)

				}

				// Check if any row was updated

				// Insert withdrawal log
				state := "ฝาก"
				headWord := "✅ ยืนยันการเติมเงิน"
				if result[0] == '-' {
					state = "ถอน"
					headWord = "💰 ยืนยันการถอนเงิน 💰"
				}

				wdLogQuery := `INSERT INTO wd (UID, STATE, NAME, AMOUNT, note) VALUES (?, ?, ?, ?, ?)`
				err = ExecuteQuery(ctx, wdLogQuery, accountNumber, state, idName, result, "LID")
				if err != nil {
					log.Printf("Error inserting withdrawal log: %v", err)

				}

				// Fetch updated credit
				var credit float64
				err = db.QueryRowContext(ctx, `SELECT Credit FROM user_data WHERE Number = ?`, accountNumber).Scan(&credit)
				if err != nil {
					log.Printf("Error fetching updated credit: %v", err)

				}

				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					headWord,
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   headWord,
									Weight: "bold",
									Size:   "xxs",
									Color:  "#1DB446",
									Margin: "xs",
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้รับเงิน:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  fmt.Sprintf("%v(%v)", idName, matches[1]),
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "จำนวน:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(result),
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "เครดิตรวม:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)),
											Size:  "xs",
											Color: "#1DB446",
											Align: "end",
										},
									},
								},
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#ffffff",
									Margin: "sm",
								},
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ทำรายการสำเร็จ",
									Weight: "bold",
									Size:   "xxs",
									Color:  "#1DB446",
									Margin: "xs",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#222222",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

			} else if len(runes) >= 4 && string(runes[:1]) == "$" && (strings.Contains(rawMessage, "ด") || strings.Contains(rawMessage, "ง")) {
				// Process valid URLs or responses from the Python script
				re := regexp.MustCompile(`^\$(\d+)\s*([ดง])\s+(\d+/\d+)\s+(\d+/\d+)\s+([ดง])(\d+)$`)

				matches := re.FindStringSubmatch(rawMessage)
				// if len(matches) != 3 {
				// 	return ""
				// }
				if matches != nil {
					fmt.Printf("Input: %s\n", rawMessage)
					fmt.Printf("ID: %s\n", matches[1])
					fmt.Printf("Side: %s\n", matches[2])
					fmt.Printf("Fraction 1: %s\n", matches[3])
					fmt.Printf("Fraction 2: %s\n", matches[4])
					fmt.Printf("Play Side: %s\n", matches[5])
					fmt.Printf("Amount: %s\n", matches[6])
					fmt.Println()
				} else {
					fmt.Printf("Input: %s - No match\n", rawMessage)
					return ""
				}
				// Extract account number
				accountNumber, err := strconv.Atoi(matches[1])
				if err != nil {
					return "ทำรายการไม่สำเร็จ ID ไม่ถูกรูปแบบ"
				}
				parts := strings.Split(matches[3], "/")
				if len(parts) != 2 {
					return ""
				}
				numerator, _ := strconv.ParseFloat(parts[0], 64)
				denominator, _ := strconv.ParseFloat(parts[1], 64)

				redRate = numerator / denominator
				parts2 := strings.Split(matches[4], "/")
				if len(parts2) != 2 {
					return ""
				}
				numerator2, _ := strconv.ParseFloat(parts2[0], 64)
				denominator2, _ := strconv.ParseFloat(parts2[1], 64)

				blueRate = numerator2 / denominator2
				// Extract amount and remove commas

				// amountStr := strings.ReplaceAll(matches[3], ",", "")
				// // amount, err := strconv.ParseFloat(amountStr, 64)
				// if matches[2] == "-" {
				// 	amountStr2, _ := strconv.Atoi(amountStr)
				// 	amountStr = fmt.Sprintf("-%v", amountStr2)
				// }
				// if err != nil {
				// 	return "ชุดข้อมูลไม่ถูกต้อง"
				// }
				// result := amountStr
				ctx := context.Background()
				// fmt.Println("Valid response received:", result)
				// if result == "0" {

				// }
				var idNum, idName string // Declare the variable to store the result

				// Execute the query
				_ = db.QueryRowContext(ctx, `SELECT ID, Name FROM user_data WHERE Number = ?`, accountNumber).Scan(&idNum, &idName)

				// query := `
				// 		UPDATE user_data
				// 		SET Credit = Credit + ?
				// 		WHERE Name = ?
				// 		AND (SELECT COUNT(*) FROM user_data WHERE Name = ?) = 1`
				// err = ExecuteQuery(ctx, query, result, idName, idName)
				// if err != nil {
				// 	log.Printf("Error updating user data: %v", err)

				// }

				// Check if any row was updated

				// Insert withdrawal log
				// state := "ฝาก"
				// headWord := "✅ ยืนยันการเติมเงิน"
				// if result[0] == '-' {
				// 	state = "ถอน"
				// 	headWord = "💰 ยืนยันการถอนเงิน 💰"
				// }

				// wdLogQuery := `INSERT INTO wd (UID, STATE, NAME, AMOUNT, note) VALUES (?, ?, ?, ?, ?)`
				// err = ExecuteQuery(ctx, wdLogQuery, accountNumber, state, idName, result, "LID")
				// if err != nil {
				// 	log.Printf("Error inserting withdrawal log: %v", err)

				// }

				// // Fetch updated credit
				var credit float64
				err = db.QueryRowContext(ctx, `SELECT Credit FROM user_data WHERE Number = ?`, accountNumber).Scan(&credit)
				if err != nil {
					log.Printf("Error fetching updated credit: %v", err)

				}
				var sideWins int
				if matches[2] == "ด" {
					sideWins = -1
				} else if matches[2] == "ง" {
					sideWins = 1
				} else {
					sideWins = 0
				}
				headWord := fmt.Sprintf("ยัดข้อมูลให้ %s สำเร็จ ", idName)
				vls := betOver(idNum, matches[5]+matches[6], int64(credit), idName, redRate, blueRate, sideWins)
				// Prepare Flex message
				flexMessage := linebot.NewFlexMessage(
					headWord,
					&linebot.BubbleContainer{
						Size: "kilo",
						Body: &linebot.BoxComponent{
							Type:   "box",
							Layout: "vertical",
							Contents: []linebot.FlexComponent{
								&linebot.TextComponent{
									Type:   "text",
									Text:   headWord,
									Weight: "bold",
									Size:   "xxs",
									Color:  "#1DB446",
									Margin: "xs",
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{
										&linebot.TextComponent{
											Type:  "text",
											Text:  "ผู้รับเงิน:",
											Size:  "xs",
											Color: "#ffffff",
										},
										&linebot.TextComponent{
											Type:  "text",
											Text:  fmt.Sprintf("%v(%v)", idName, matches[1]),
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								&linebot.BoxComponent{
									Type:   "box",
									Layout: "horizontal",
									Contents: []linebot.FlexComponent{

										&linebot.TextComponent{
											Type:  "text",
											Text:  matches[5] + vls,
											Size:  "xs",
											Color: "#ffffff",
											Align: "end",
										},
									},
								},
								// &linebot.BoxComponent{
								// 	Type:   "box",
								// 	Layout: "horizontal",
								// 	Contents: []linebot.FlexComponent{
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  "เครดิตรวม:",
								// 			Size:  "xs",
								// 			Color: "#ffffff",
								// 		},
								// 		&linebot.TextComponent{
								// 			Type:  "text",
								// 			Text:  formatWithCommas(fmt.Sprintf("%.0f", credit)),
								// 			Size:  "xs",
								// 			Color: "#1DB446",
								// 			Align: "end",
								// 		},
								// 	},
								// },
								&linebot.SeparatorComponent{
									Type:   "separator",
									Color:  "#ffffff",
									Margin: "sm",
								},
								&linebot.TextComponent{
									Type:   "text",
									Text:   "✅ ทำรายการสำเร็จ",
									Weight: "bold",
									Size:   "xxs",
									Color:  "#1DB446",
									Margin: "xs",
									Align:  "center",
								},
							},
							Spacing:         "sm",
							BackgroundColor: "#222222",
						},
					},
				)

				// Send reply message
				_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
				if err != nil {
					log.Printf("Error sending reply message: %v", err)
				}

			}

		}

		switch rawMessage {
		case "S2":
			{
				te := summarize4("R", round)
				// flexMessage, _ := GenerateFlexMessage(te)
				// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
				// // playerData, err := parsePlayerData(input)
				// if err != nil {
				// 	fmt.Println("Error parsing input:", err)
				// 	return input
				// }

				// Generate Flex message
				// flexMessage, _ := GenerateFlexMessage4(te, localRound, "แดงชนะ")
				// err = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				// if err != nil {
				// 	log.Println("Error sending Flex message: %v", err)
				// }

				// log.Println("Flex message sent successfully")
				// te2 := fmt.Sprintf("%v", te)
				// nextRound()
				flexMessage, _ := GenerateFlexMessage22(te, localRound, "OK")
				// _ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}
				te2 := fmt.Sprintf("%v", te)
				return te2
			}
		case "S3":
			{
				te := summarize4("R", round)
				// flexMessage, _ := GenerateFlexMessage(te)
				// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
				// // playerData, err := parsePlayerData(input)
				// if err != nil {
				// 	fmt.Println("Error parsing input:", err)
				// 	return input
				// }

				// Generate Flex message
				// flexMessage, _ := GenerateFlexMessage4(te, localRound, "แดงชนะ")
				// err = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				// if err != nil {
				// 	log.Println("Error sending Flex message: %v", err)
				// }

				// log.Println("Flex message sent successfully")
				// te2 := fmt.Sprintf("%v", te)
				// nextRound()
				flexMessage, _ := GenerateFlexMessage22V2(te, localRound, "OK")
				// _ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}
				te2 := fmt.Sprintf("%v", te)
				return te2
			}
		case "R":
			if localState == 0 {
				ctx := context.Background()
				mes, _ := Reverse(ctx, localState, localRound)
				return mes
			}

			return "ต้องประกาศผลก่อน"
		case "Sด":
			if localState == 0 {
				te := Summarize2p("R", round)
				// flexMessage, _ := GenerateFlexMessage(te)
				// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
				// // playerData, err := parsePlayerData(input)
				// if err != nil {
				// 	fmt.Println("Error parsing input:", err)
				// 	return input
				// }

				// Generate Flex message
				flexMessage, _ := GenerateFlexMessage2p(te, localRound, "แดงชนะ")
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")
				te2 := fmt.Sprintf("%v", te)
				// nextRound()

				return te2
			}
			return "ไม่สามารถแสดงผลเพิ่มเติมหลังเปิดรอบใหม่"
		case "Sส":
			if localState == 0 {
				te := Summarize2p("S", round)
				// flexMessage, _ := GenerateFlexMessage(te)
				// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
				// // playerData, err := parsePlayerData(input)
				// if err != nil {
				// 	fmt.Println("Error parsing input:", err)
				// 	return input
				// }

				// Generate Flex message
				flexMessage, _ := GenerateFlexMessage2p(te, localRound, "เสมอ")
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")
				te2 := fmt.Sprintf("%v", te)
				// nextRound()

				return te2
			}
			return "ไม่สามารถแสดงผลเพิ่มเติมหลังเปิดรอบใหม่"
		case "Sง":
			if localState == 0 {
				te := Summarize2p("B", round)
				// flexMessage, _ := GenerateFlexMessage(te)
				// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
				// // playerData, err := parsePlayerData(input)
				// if err != nil {
				// 	fmt.Println("Error parsing input:", err)
				// 	return input
				// }

				// Generate Flex message
				flexMessage, _ := GenerateFlexMessage2p(te, localRound, "น้ำเงินชนะ")
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")
				te2 := fmt.Sprintf("%v", te)
				// nextRound()

				return te2
			}
			return "ไม่สามารถแสดงผลเพิ่มเติมหลังเปิดรอบใหม่"

		case "E":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }
			setState(2)
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			return te2
		case "(prohibited)":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }
			setState(2)
			// Generate Flex message
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}
				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			return te2
		case "ปด":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }
			setState(2)
			// Generate Flex message
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			return te2
		case "ป":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }
			setState(2)
			// Generate Flex message
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			return te2
		case "(x mark)":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }
			setState(2)
			// Generate Flex message
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			return te2
		case "ปิด":
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if localState == 2 || localState == 0 {
				return ""
			}

			te := Summarize("R", round)
			// flexMessage, _ := GenerateFlexMessage(te)
			// input := "[[ชื่อ  ได้เสีย  คงเหลือ] [(6)James JR  -11  คงเหลือ 104088] [(14)James JR2  1  คงเหลือ -2006]]"
			// // playerData, err := parsePlayerData(input)
			// if err != nil {
			// 	fmt.Println("Error parsing input:", err)
			// 	return input
			// }

			// Generate Flex message
			if showE {
				flexMessage, _ := GenerateFlexMessage(te)
				bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
				if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
					log.Println(err)
				}

				log.Println("Flex message sent successfully")

			}
			// Generate Flex message
			te2 := fmt.Sprintf("")

			setState(2)
			return te2
		case "S":
			st, _ := summarize3("x", localRound)
			fmt.Println("XX", st)
			flexMessage, _ := GenerateFlexMessageR(st)

			// SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
			if _, err := bot.ReplyMessage(replyToken, flexMessage...).Do(); err != nil {
				log.Println(err)
			}
			return st
		case "ยกเลิกรอบ":

			return deniedSub(db, localRound, localSub, localState)
		case "U":
			a, b, c, d, _ := getTeamSumsAndTotalDeposit(localRound)
			SendFlexMessage(replyToken, IllustrateU(int64(a), int64(b), int64(c), int64(d)), LineChannelAccessToken)
			return ""
		case "RE":
			reRound()
			lineBotAPI, err := linebot.New(LineChannelSecret, LineChannelAccessToken)
			if err != nil {
				log.Println(err)
			}

			// Create the Flex message structure with only the hero image and action
			flexMessage := linebot.NewFlexMessage(
				"รีรอบการเดิมพันสำเร็จ",
				&linebot.BubbleContainer{
					Type: linebot.FlexContainerTypeBubble,
					Size: "hecto",
					Body: &linebot.BoxComponent{
						Type:            linebot.FlexComponentTypeBox,
						Layout:          linebot.FlexBoxLayoutTypeVertical,
						Spacing:         "sm",
						BackgroundColor: "#1C1C1C", // พื้นหลังสีดำอมเทา
						Contents: []linebot.FlexComponent{
							&linebot.BoxComponent{
								Type:   linebot.FlexComponentTypeBox,
								Layout: linebot.FlexBoxLayoutTypeVertical,
								Contents: []linebot.FlexComponent{
									&linebot.ImageComponent{
										Type: linebot.FlexComponentTypeImage,
										URL:  "https://png.pngtree.com/png-vector/20221215/ourmid/pngtree-green-check-mark-png-image_6525691.png",
										Size: "xs",
									},
								},
							},
							&linebot.BoxComponent{
								Type:   linebot.FlexComponentTypeBox,
								Layout: linebot.FlexBoxLayoutTypeVertical,
								Contents: []linebot.FlexComponent{
									&linebot.BoxComponent{
										Type:           linebot.FlexComponentTypeBox,
										Layout:         linebot.FlexBoxLayoutTypeVertical,
										JustifyContent: linebot.FlexComponentJustifyContentTypeSpaceBetween,
										AlignItems:     linebot.FlexComponentAlignItemsTypeCenter,
										Contents: []linebot.FlexComponent{

											&linebot.BoxComponent{
												Type:   linebot.FlexComponentTypeBox,
												Layout: linebot.FlexBoxLayoutTypeVertical,
												Contents: []linebot.FlexComponent{
													&linebot.TextComponent{
														Type:  linebot.FlexComponentTypeText,
														Text:  "[ รีรอบการเดิมพัน สำเร็จ ]",
														Color: "#FFC107", // ทองเข้ม
														Size:  "sm",
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Styles: &linebot.BubbleStyle{
						Footer: &linebot.BlockStyle{
							Separator: true,
						},
					},
				},
			)

			// Replace 'your-reply-token' with the actual reply token you receive from LINE messaging API
			replyToken := replyToken

			// Send the flex message to the user
			_, err = lineBotAPI.ReplyMessage(replyToken, flexMessage).Do()
			if err != nil {
				log.Println(err)
			}

			// SendFlexMessage()
			return "รีรอบสำเร็จ^^"
		case "หลังบ้าน":
			SendFlexMessage(replyToken, GenerateFlexHome(), LineChannelAccessToken)
			return "รีรอบสำเร็จ^^"
		case "GAME":
			SendFlexMessage(replyToken, GenerateFlexHome2(), LineChannelAccessToken)
			return "รีรอบสำเร็จ^^"

		case "C":

			// InsertUser(userID, displayName, pictureURL)
			redPredict, bluePredict := predictRB(userID, localRound)
			c1, c2, cs := getBalance(userID), getBalance2(userID), getBalance(userID)-getBalance2(userID)
			flexMessage, _ := GenerateFlexC(fmt.Sprintf("%v", num), displayName, strconv.FormatInt(cs, 10), strconv.FormatInt(c2, 10), strconv.FormatInt(redPredict, 10), strconv.FormatInt(bluePredict, 10), strconv.Itoa(localRound), pictureURL, strconv.FormatInt(redPredict+c1, 10), strconv.FormatInt(bluePredict+c1, 10), strconv.FormatInt(redPredict+cs, 10), strconv.FormatInt(bluePredict+cs, 10))
			_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			return "C is you too"
		case "c":

			// InsertUser(userID, displayName, pictureURL)
			redPredict, bluePredict := predictRB(userID, localRound)
			c1, c2, cs := getBalance(userID), getBalance2(userID), getBalance(userID)-getBalance2(userID)
			flexMessage, _ := GenerateFlexC(fmt.Sprintf("%v", num), displayName, strconv.FormatInt(cs, 10), strconv.FormatInt(c2, 10), strconv.FormatInt(redPredict, 10), strconv.FormatInt(bluePredict, 10), strconv.Itoa(localRound), pictureURL, strconv.FormatInt(redPredict+c1, 10), strconv.FormatInt(bluePredict+c1, 10), strconv.FormatInt(redPredict+cs, 10), strconv.FormatInt(bluePredict+cs, 10))
			_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			return "C is you too"
		case "cc":
			_, _, num, _ := GetUserData(userID)
			redPredict, bluePredict := predictRB(userID, localRound)
			// cs := getBalance(userID) - getBalance2(userID)
			flexMessage, _ := GenerateFlexMessageC2(SummarizeC2(userID, displayName), fmt.Sprintf("(%v)%v", num, displayName), pictureURL, userID, fmt.Sprintf("%d", redPredict), fmt.Sprintf("%d", bluePredict))
			_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			return "a"
		case "Cc":
			_, _, num, _ := GetUserData(userID)
			redPredict, bluePredict := predictRB(userID, localRound)
			// cs := getBalance(userID) - getBalance2(userID)
			flexMessage, _ := GenerateFlexMessageC2(SummarizeC2(userID, displayName), fmt.Sprintf("(%v)%v", num, displayName), pictureURL, userID, fmt.Sprintf("%d", redPredict), fmt.Sprintf("%d", bluePredict))
			_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			return "a"
		case "D":

			if err != nil {
				fmt.Println("Error:", err)
				return displayName
			}
			resultString := fmt.Sprintf("Data retrieved successfully for user ID %s: credit=%d, credit2=%d, num=%d", userID, credit, credit2, num)
			return resultString
		case "B":

			if err != nil {
				fmt.Println("Error:", err)
				return "Something is being calculated"
			}

			// Print the results
			fmt.Println("Local Round:", localRound)
			fmt.Println("Local Sub:", localSub)
			fmt.Println("Local State:", localState)
			fmt.Printf("Local Red Rate: %.2f\n", localRedRate)   // Printing the parsed value of the fraction
			fmt.Printf("Local Blue Rate: %.2f\n", localBlueRate) // Printing the parsed value of the fraction
			fmt.Println("Local Red Open:", localRedOpen)
			fmt.Println("Local Blue Open:", localBlueOpen)
			fmt.Println("Local Command:", localCommand)
			fmt.Println("Local Min:", localMin)
			fmt.Println("Local Max:", localMax)
			fmt.Println("Local Win:", localWin)
			return "Something is being calculated"
		default:
			localRound, localSub, _, _, _, _,
				_, _, _, _, _, err := GetLocalVar()

			if err != nil {
				fmt.Println("Error:", err)
				return "Something is being calculated"
			}
			pattern1 := `([ดง])\s*(\d+/\d+)\s*(\d+/\d+)\s*=\s*(\d+)`
			pattern2 := `([ดง])\s*(\d+/\d+)\s*รอง\p{Thai}+\s*=\s*(\d+)`
			pattern3 := `(ส[ด|ง])\s*(\d+/\d+)\s*=\s*(\d+)`
			pattern4 := `(ตร)\s*(\d+/\d+)?\s*=\s*(\d+)`
			pattern5 := `10/10\s*=\s*(\d{2,7})`
			pattern6 := `([ดงสตร]?)\s*(\d+/?\d+)?\s*[\p{Thai}]*\s*=\s*(\d+)?`
			pattern2N := `([ดง])\s*(\d+/\d+)\s*ต่อ\p{Thai}+\s*=\s*(\d+)`
			if localState == 1 {
				return ""
			}

			// Compile the patterns
			r1 := regexp.MustCompile(pattern1)
			r2 := regexp.MustCompile(pattern2)
			r2N := regexp.MustCompile(pattern2N)
			r3 := regexp.MustCompile(pattern3)
			r4 := regexp.MustCompile(pattern4)
			r5 := regexp.MustCompile(pattern5)
			r6 := regexp.MustCompile(pattern6)
			fmt.Println("Pattern 1 matched!")
			// fmt.Println(r5)
			rawMessage = strings.ReplaceAll(rawMessage, ",", "")
			rawMessage2 := strings.ReplaceAll(rawMessage, "รับ", "=")
			rawMessage2 = strings.TrimSpace(rawMessage2)
			rawMessage = strings.TrimSpace(rawMessage)
			// Check each pattern and handle accordingly
			if groupID == groupC || groupID == GroupT {
				return ""
			}
			if r1.MatchString(rawMessage) {
				fmt.Println("Pattern 1 matched!")
				matches := r1.FindStringSubmatch(rawMessage)
				blue := matches[2]
				red := matches[3]
				high := matches[4]
				// Split fractions
				numerator, denominator := splitFraction(blue)
				numerator2, denominator2 := splitFraction(red)
				if denominator == 0 || denominator2 == 0 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง" // Return error message for invalid rates
				}

				// Perform the division
				result := numerator / denominator
				result2 := numerator2 / denominator2
				fmt.Println(result, result2, result > result2, matches[0], "R", matches[1])
				if result <= result2 && matches[1] == "ง" {
					blue = matches[3]
					red = matches[2]
					fmt.Println(result, result2, result > result2)
				} else if result2 < result && matches[1] == "ง" {
					blue = matches[2]
					red = matches[3]
					fmt.Println(result, result2, result2 > result)
				}
				if result >= result2 && matches[1] == "ด" {
					blue = matches[3]
					red = matches[2]
					fmt.Println(result, result2, result > result2)
				} else if result2 < result && matches[1] == "ด" {
					blue = matches[2]
					red = matches[3]
					fmt.Println(result, result2, result2 > result)
				}

				// Save to the data map
				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": getColor3(matches[1]),
					"SB":    "1",
					"SR":    "1",
					"win":   getWin(matches[1]),
				}
				localSub := localSub + 1                        // Incremented local_sub value
				localState := 1                                 // Local state value (int)
				localWin, _ := strconv.Atoi(getWin(matches[1])) // Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)            // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string)          // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)

				// Print or execute the SQL query
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ExecuteQuery(ctx, updateQuery)
				if showO {
					newM := getMessage(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}

			} else if r2.MatchString(rawMessage) {
				fmt.Println("Pattern 2 matched!")
				matches := r2.FindStringSubmatch(rawMessage)
				blue := matches[2]
				red := matches[2]
				high := matches[3]
				parts := strings.Split(matches[2], "/")
				if len(parts) != 2 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง"
				}

				// Parse numerator and denominator
				_, err1 := strconv.Atoi(parts[0])
				denominator, err2 := strconv.Atoi(parts[1])

				if err1 != nil || err2 != nil || denominator == 0 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง" // Handle invalid input or division by zero
				}

				// Optionally, return the normalized fraction

				// Similar logic as pattern 1 but with different mapping

				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": getColor3(matches[1]),
					"SB":    getSB(matches[1]),
					"SR":    getSR(matches[1]),
					"win":   getWin(matches[1]),
				}
				localSub := localSub + 1                        // Incremented local_sub value
				localState := 1                                 // Local state value (int)
				localWin, _ := strconv.Atoi(getWin(matches[1])) // Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)            // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string)          // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)

				// Print or execute the SQL query
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ExecuteQuery(ctx, updateQuery)
				if showO {
					newM := getMessage3(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage3(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}

			} else if r2N.MatchString(rawMessage) {
				fmt.Println("Pattern 2N matched!")
				matches := r2N.FindStringSubmatch(rawMessage)
				blue := matches[2]
				red := matches[2]
				high := matches[3]
				parts := strings.Split(matches[2], "/")
				if len(parts) != 2 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง"
				}

				// Parse numerator and denominator
				_, err1 := strconv.Atoi(parts[0])
				denominator, err2 := strconv.Atoi(parts[1])

				if err1 != nil || err2 != nil || denominator == 0 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง" // Handle invalid input or division by zero
				}

				// Optionally, return the normalized fraction

				// Similar logic as pattern 1 but with different mapping

				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": getColor3(matches[1]),
					"SB":    getSR(matches[1]),
					"SR":    getSB(matches[1]),
					"win":   getWin(matches[1]),
				}
				localSub := localSub + 1                        // Incremented local_sub value
				localState := 1                                 // Local state value (int)
				localWin, _ := strconv.Atoi(getWin(matches[1])) // Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)            // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string)          // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)

				// Print or execute the SQL query
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ExecuteQuery(ctx, updateQuery)
				if showO {
					newM := getMessage3(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage3(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}

			} else if r3.MatchString(rawMessage) {
				fmt.Println("Pattern 3 matched!")
				matches := r3.FindStringSubmatch(rawMessage)
				parts := strings.Split(matches[2], "/")
				if len(parts) != 2 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง"
				}

				// Parse numerator and denominator
				_, err1 := strconv.Atoi(parts[0])
				denominator, err2 := strconv.Atoi(parts[1])

				if err1 != nil || err2 != nil || denominator == 0 {
					return "รูปแบบการเปิดราคาไม่ถูกต้อง" // Handle invalid input or division by zero
				}

				// Optionally, return the normalized fraction

				blue := getBlueForPattern3(matches[1], matches[2])
				red := getRedForPattern3(matches[1], matches[2])
				high := matches[3]

				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": getColor3(matches[1]),
					"SB":    "1",
					"SR":    "1",
					"win":   getWin(matches[1]),
				}
				localSub := localSub + 1                        // Incremented local_sub value
				localState := 1                                 // Local state value (int)
				localWin, _ := strconv.Atoi(getWin(matches[1])) // Local win value (int, make sure getWin() returns an integer)
				// Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)   // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string) // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)

				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ExecuteQuery(ctx, updateQuery)
				if showO {
					newM := getMessage(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}

			} else if r4.MatchString(rawMessage2) {
				fmt.Println("Pattern 4 matched!")
				matches := r4.FindStringSubmatch(rawMessage2)
				blue, red, high := handlePattern4(matches)

				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": getColor3(matches[1]),
					"SB":    "1",
					"SR":    "1",
					"win":   "0",
				}
				localSub := localSub + 1               // Incremented local_sub value
				localState := 1                        // Local state value (int)
				localWin := 0                          // Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)   // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string) // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)

				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				ExecuteQuery(ctx, updateQuery)

				if showO {
					newM := getMessage2(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage2(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}
			} else if r5.MatchString(rawMessage) {
				fmt.Println("Pattern 5 matched!")
				matches := r5.FindStringSubmatch(rawMessage)
				high := matches[1]
				blue := "10/10"
				red := "10/10"

				data := map[string]interface{}{
					"blue":  blue,
					"red":   red,
					"high":  high,
					"color": "1",
					"SB":    "1",
					"SR":    "1",
					"win":   "0",
				}
				localSub := localSub + 1               // Incremented local_sub value
				localState := 1                        // Local state value (int)
				localWin := 0                          // Local win value (int, make sure getWin() returns an integer)
				localRedRate := data["red"].(string)   // Ensure it's a string for varchar
				localBlueRate := data["blue"].(string) // Ensure it's a string for varchar

				// Convert SR and SB from string to int
				localRedOpen, err := convertToInt(data["SR"])
				if err != nil {
					fmt.Println("Error converting SR:", err)
				}

				localBlueOpen, err := convertToInt(data["SB"])
				if err != nil {
					fmt.Println("Error converting SB:", err)
				}

				localCommand := rawMessage // Should be a string
				localMin := 1              // Make sure it's an integer

				// Convert high (assuming it's a string) to int
				localMax, err := convertToInt(data["high"])
				if err != nil {
					fmt.Println("Error converting high:", err)
				}

				// Update query using correct types
				updateQuery := fmt.Sprintf(`
				UPDATE environment 
				SET 
				local_round='%d',
				local_sub='%d',
				local_state='%d',
				local_win='%d',
				local_red_rate='%s',
				local_blue_rate='%s',
				local_red_open='%d',
				local_blue_open='%d',
				local_command='%s',
				local_min='%d',
				local_max='%d'
				WHERE 1
				`,
					localRound,    // local_round (int)
					localSub,      // local_sub (int)
					localState,    // local_state (int)
					localWin,      // local_win (int)
					localRedRate,  // local_red_rate (varchar)
					localBlueRate, // local_blue_rate (varchar)
					localRedOpen,  // local_red_open (int)
					localBlueOpen, // local_blue_open (int)
					localCommand,  // local_command (varchar)
					localMin,      // local_min (int)
					localMax,      // local_max (int)
				)
				ctx := context.Background()
				// Print or execute the SQL query
				fmt.Println(updateQuery)
				err6 := ExecuteQuery(ctx, updateQuery) // local_max (int)

				if err6 != nil {
					fmt.Println("Error:", err6.Error()) // Print the error message
				}
				if showO {
					newM := getMessage2(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
					if showOP {
						threeLine := getMessage2(localWin, localRedRate, localBlueRate, strconv.Itoa(localMax))
						messageParts := strings.Split(threeLine, "\n")
						fmt.Println("First Line:", messageParts[0])
						fmt.Println("Second Line:", messageParts[1])
						// SendFlexMessage(replyToken, GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1]), LineChannelAccessToken, rawMessage)
						// messages :=GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])
						bot, _ := linebot.New(LineChannelSecret, LineChannelAccessToken)
						flexMessage := GenerateFlexOPEN(rawMessage, messageParts[0], messageParts[1])

						// Send the reply message
						if _, err := bot.ReplyMessage(replyToken, flexMessage).Do(); err != nil {
							log.Println(err)
						}
						return ""

					} else {
						return newM
					}
				} else {
					return ""
				}

			} else if r6.MatchString(rawMessage) {
				return ""
				// return "รูปแบบไม่ถูกต้อง เปิดราคาไม่สำเร็จ"
			}
			return ""
		}

	} else {

		switch strings.ToLower(strings.Trim(rawMessage, " ")) {

		case "c":
			if 1 == 1+0 {
				// InsertUser(userID, displayName, pictureURL)
				_, _, num64, _ := GetUserData(userID)
				num := int(num64)
				redPredict, bluePredict := predictRB(userID, localRound)

				c1, c2, cs := getBalance(userID), getBalance2(userID), getBalance(userID)-getBalance2(userID)
				flexMessage, _ := GenerateFlexC(strconv.Itoa(num), displayName, strconv.FormatInt(cs, 10), strconv.FormatInt(c2, 10), strconv.FormatInt(redPredict, 10), strconv.FormatInt(bluePredict, 10), strconv.Itoa(localRound), pictureURL, strconv.FormatInt(redPredict+c1, 10), strconv.FormatInt(bluePredict+c1, 10), strconv.FormatInt(redPredict+cs, 10), strconv.FormatInt(bluePredict+cs, 10))
				_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				return "C is you too"
			} else {
				// InsertUser(userID, displayName, pictureURL)
				// redPredict, bluePredict := predictRB(userID, localRound)
				// c1, c2, cs := getBalance(userID), getBalance2(userID), getBalance(userID)-getBalance2(userID)
				// flexMessage, _ := GenerateFlexC(strconv.Itoa(num), displayName, strconv.FormatInt(cs, 10), strconv.FormatInt(c2, 10), strconv.FormatInt(redPredict, 10), strconv.FormatInt(bluePredict, 10), strconv.Itoa(localRound), pictureURL, strconv.FormatInt(redPredict+c1, 10), strconv.FormatInt(bluePredict+c1, 10), strconv.FormatInt(redPredict+cs, 10), strconv.FormatInt(bluePredict+cs, 10))
				// _ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
				return ""
			}
		case "cc":
			_, _, num, _ := GetUserData(userID)
			redPredict, bluePredict := predictRB(userID, localRound)
			// cs := getBalance(userID) - getBalance2(userID)
			flexMessage, _ := GenerateFlexMessageC2(SummarizeC2(userID, displayName), fmt.Sprintf("(%v)%v", num, displayName), pictureURL, userID, fmt.Sprintf("%v", redPredict), fmt.Sprintf("%v", bluePredict))
			_ = SendFlexMessage(replyToken, flexMessage, LineChannelAccessToken)
			return "a"

		default:
			if localState == 1 && (groupID == groupPlay) {
				rawMessage = strings.TrimSpace(rawMessage)
				words := strings.Fields(rawMessage)

				// Pattern ที่ยอมรับได้: ด100 หรือ ง100 (ต้องขึ้นต้นด้วย "ด" หรือ "ง" และตามด้วยเลขเท่านั้น)
				validWordPattern := regexp.MustCompile(`^(ด|ง)[0-9]{1,5}$`)

				// Pattern ต้องห้าม: มีตัวแปลก เช่น - / ' หรือคำว่า แดง น้ำเงิน เงิน หรือผสมดง งด
				invalidPattern := regexp.MustCompile(`[\/\'\-]|แดง|น้ำเงิน|เงิน|ดง|งด`)

				for _, word := range words {
					// ถ้าเป็นคำพูดยาวเกิน 10 ตัวอักษร → ข้าม (พิจารณาว่าเป็นข้อความทั่วไป)
					if len([]rune(word)) > 10 {
						continue
					}
					// ถ้าคำมี pattern ที่ไม่อนุญาต หรือไม่ match รูปแบบที่ควร → แจ้งไม่ติด
					if invalidPattern.MatchString(word) || !validWordPattern.MatchString(word) {
						return "❌ไม่ติด❌"
					}
				}
			}
			return ""
		}
	}
}
func nextRound() {
	updateQuery := fmt.Sprintf(`
    UPDATE environment 
    SET 
        local_round = local_round + 1,
        local_sub = 1,
		local_state = 0
    WHERE 1
`)

	// Print or execute the SQL query
	ctx := context.Background()
	// Print or execute the SQL query
	fmt.Println(updateQuery)
	ExecuteQuery(ctx, updateQuery)

}
func reRound() {
	updateQuery := fmt.Sprintf(`
    UPDATE environment 
    SET 
        local_round = 1,
        local_sub = 1,
		local_state = 0
    WHERE 1
`)

	// Print or execute the SQL query
	ctx := context.Background()
	// Print or execute the SQL query
	fmt.Println(updateQuery)
	ExecuteQuery(ctx, updateQuery)
	updateQuery2 := fmt.Sprintf(`
    UPDATE playinglog 
    SET 
        Game_play = 'END'
    WHERE Game_play != 'END'
`)

	// Print or execute the SQL query
	ctx2 := context.Background()
	// Print or execute the SQL query
	fmt.Println(updateQuery2)
	ExecuteQuery(ctx2, updateQuery2)
	updateQuery3 := fmt.Sprintf(`
    UPDATE wd
    SET checks =1 
    WHERE checks = 0
`)

	// Print or execute the SQL query
	ctx3 := context.Background()
	// Print or execute the SQL query
	fmt.Println(updateQuery3)
	ExecuteQuery(ctx3, updateQuery3)
	updateQuery4 := fmt.Sprintf(`
    UPDATE stats
    SET STATUS ="END"
    WHERE STATUS != "END"
`)

	// Print or execute the SQL query
	ctx4 := context.Background()
	// Print or execute the SQL query
	fmt.Println(updateQuery4)
	ExecuteQuery(ctx4, updateQuery4)

}

// Helper function to format numbers with commas
func formatWithCommas(value interface{}) string {
	var number int64
	var err error

	switch v := value.(type) {
	case int:
		number = int64(v)
	case int64:
		number = v
	case string:
		number, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Printf("Error converting string to int64: %v", err)
			return v // Return the original string if conversion fails
		}
	default:
		log.Printf("Unsupported type: %T", value)
		return fmt.Sprintf("%v", value) // Return the original value if unsupported type
	}

	// Format the number with commas manually
	return formatNumberWithCommas(number)
}

// formatNumberWithCommas inserts commas into an integer
func formatNumberWithCommas(number int64) string {
	// Handle negative numbers
	isNegative := number < 0
	if isNegative {
		number = -number // Work with the absolute value
	}

	// Convert the number to a string
	str := strconv.FormatInt(number, 10)
	if len(str) <= 3 {
		if isNegative {
			return "-" + str
		}
		return str
	}

	// Use a strings.Builder to insert commas
	var sb strings.Builder
	n := len(str)
	for i, ch := range str {
		if i > 0 && (n-i)%3 == 0 {
			sb.WriteRune(',')
		}
		sb.WriteRune(ch)
	}

	// Add the negative sign back if the number was negative
	if isNegative {
		return "-" + sb.String()
	}
	return sb.String()
}

func splitFraction(fraction string) (float64, float64) {
	parts := regexp.MustCompile(`/`).Split(fraction, -1)
	numerator, _ := strconv.ParseFloat(parts[0], 64)
	denominator, _ := strconv.ParseFloat(parts[1], 64)
	return numerator, denominator
}

// Helper function to get color based on message
func getColor3(message string) string {
	if message == "ง" {
		return "1"
	}
	return "-1"
}

// Helper function to determine win based on message
func getWin(message string) string {
	if message == "ง" {
		return "1"
	} else if message == "สง" {
		return "1"
	}
	return "-1"
}

// Helper function for SB value based on message
func getSB(message string) string {
	if message == "ด" {
		return "1"
	}
	return "0"
}

// Helper function for SR value based on message
func getSR(message string) string {
	if message == "ด" {
		return "0"
	}
	return "1"
}

// Helper function to handle pattern 3
func getBlueForPattern3(message, fraction string) string {
	if message == "ง" {
		return fraction
	} else if message == "สง" {
		return fraction
	}
	return "10/10"
}

func getRedForPattern3(message, fraction string) string {
	if message == "ด" {
		return "10/10"
	} else if message == "สง" {
		return "10/10"
	}

	return fraction
}

// Helper function to handle pattern 4
func handlePattern4(matches []string) (string, string, string) {
	blue := "10/9"
	red := "10/9"
	if matches[1] == "ตร" && matches[2] != "" {
		// Use provided fraction
		blue = matches[2]
		red = matches[2]
	}
	return blue, red, matches[3]
}

//บังคับเพิ่มเพื่อน
// func getUserProfile(userID string) (string, string, string, error) {
// 	url := "https://api.line.me/v2/bot/profile/" + userID
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return "", "", "", err
// 	}

// 	req.Header.Set("Authorization", "Bearer "+LineChannelAccessToken)
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", "", "", err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 {
// 		return "", "", "", fmt.Errorf("error getting user profile: status code %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", "", "", err
// 	}

// 	var profile map[string]interface{}
// 	err = json.Unmarshal(body, &profile)
// 	if err != nil {
// 		return "", "", "", err
// 	}

// 	displayName, ok := profile["displayName"].(string)
// 	if !ok {
// 		displayName = ""
// 	}

// 	pictureURL, ok := profile["pictureUrl"].(string)
// 	if !ok {
// 		pictureURL = ""
// 	}

// 	statusMessage, ok := profile["statusMessage"].(string)
// 	if !ok {
// 		statusMessage = ""
// 	}

// 	return displayName, pictureURL, statusMessage, nil
// }

// GetGroupMemberProfile fetches the profile of a user in a group or room context
func getUserProfile(groupID, userID string) (string, string, string, error) {
	var url string
	if groupID == "" {
		// 1-on-1 chat: Use the user profile endpoint
		url = fmt.Sprintf("https://api.line.me/v2/bot/profile/%s", userID)
	} else {
		// Group chat: Use the group member profile endpoint
		url = fmt.Sprintf("https://api.line.me/v2/bot/group/%s/member/%s", groupID, userID)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", "", err
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+LineChannelAccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("error getting user profile: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	// Parse response JSON
	var profile struct {
		DisplayName   string `json:"displayName"`
		PictureURL    string `json:"pictureUrl"`
		StatusMessage string `json:"statusMessage"`
	}
	err = json.Unmarshal(body, &profile)
	if err != nil {
		return "", "", "", err
	}

	return profile.DisplayName, profile.PictureURL, profile.StatusMessage, nil
}

// betting PART
var (
	reply         string
	min           int
	max           int
	blue          string
	red           string
	high          float64
	color         string
	round         int
	subRound      int
	redRate       float64
	blueRate      float64
	redOpen       bool
	blueOpen      bool
	command       string
	win           int
	redMaxLost    int64
	redMaxProfit  int64
	blueMaxLost   int64
	blueMaxProfit int64
	state         int
)

func predictRB(userId string, round int) (int64, int64) {
	query2 := `
    SELECT 
        'red' AS team, 
        COALESCE(SUM(b1), 0) AS redMaxLost, 
        COALESCE(SUM(advanced_pay), 0) AS totalMaxLost, 
        COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
    FROM 
        playinglog 
    WHERE 
        ID = ? AND round = ? AND b1 > 0 AND Game_play != ? AND Game_play != ?
    UNION ALL
    SELECT 
        'blue' AS team, 
        COALESCE(SUM(j1), 0) AS blueMaxLost, 
        COALESCE(SUM(advanced_pay), 0) AS totalMaxLost, 
        COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
    FROM 
        playinglog 
    WHERE 
        ID = ? AND round = ? AND j1 > 0 AND Game_play NOT IN (?, ?);
    `

	// fmt.Println("Local Min:")
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, query2, userId, round, "END", "AFTER", userId, round, "END", "AFTER")
	if err != nil {
		// fmt.Println("Local Min:")
		log.Println(err)
	}
	defer rows.Close()

	// fmt.Println("Local Min:")

	var redMaxLost, blueMaxLost, redMaxProfit, blueMaxProfit int64

	// Retrieve the values for red and blue team
	for rows.Next() {
		var team string
		var sumLost int64
		var maxLost int64     // Max loss is a float64
		var maxProfit float64 // Max profit is a float64

		// Scan the values accordingly
		if err := rows.Scan(&team, &sumLost, &maxLost, &maxProfit); err != nil {
			log.Println(err)
		}

		if team == "red" {
			// Assign the sum of b1 (redMaxLost) and maxlost/maxprofit
			redMaxLost = maxLost
			redMaxProfit = int64(maxProfit) // Convert to int if necessary
			fmt.Println("Local Min:", redMaxLost, redMaxProfit)
		} else {
			// Assign the sum of j1 (blueMaxLost) and maxlost/maxprofit
			blueMaxLost = maxLost
			blueMaxProfit = int64(maxProfit) // Convert to int if necessary
			fmt.Println("Local Min:", blueMaxLost, blueMaxProfit)
		}
	}

	// Return the calculated values
	return redMaxProfit - blueMaxLost, blueMaxProfit - redMaxLost
}
func GenerateFlexOPEN(row1, row2, command string) *linebot.FlexMessage {
	firstChar := string([]rune(row1)[0])
	secondChar := string([]rune(row1)[1])
	var boxColor, row2Color, commandColor string

	switch firstChar {
	case "ด":
		boxColor = "#FF0000"
		row2Color = "#FF0000"
		commandColor = "#0000FF"
	case "ง":
		boxColor = "#0000FF"
		row2Color = "#0000FF"
		commandColor = "#FF0000"

	case "ส":
		switch secondChar {
		case "ด":
			boxColor = "#FF0000"
			row2Color = "#FF0000"
			commandColor = "#0000FF"
		case "ง":
			boxColor = "#0000FF"
			row2Color = "#0000FF"
			commandColor = "#FF0000"
		default:
			boxColor = "#00a000"
			row2Color = "#ff0000"
			commandColor = "#0000ff"

		}
	default:
		boxColor = "#00a000"
		row2Color = "#ff0000"
		commandColor = "#0000ff"
	}
	var showFullOpen string
	if showFullO {
		showFullOpen = "100%"
	} else {
		showFullOpen = "0px"
	}
	flexMessage := linebot.NewFlexMessage(
		command,
		&linebot.BubbleContainer{
			Size: "kilo", // ลดขนาดจาก mega เป็น kilo
			Hero: &linebot.BoxComponent{
				Type:   "box",
				Layout: "vertical",
				Contents: []linebot.FlexComponent{
					&linebot.BoxComponent{
						Type:            "box",
						Layout:          "vertical",
						BackgroundColor: boxColor,

						PaddingAll: "0px", // ลด Padding
						Contents: []linebot.FlexComponent{
							&linebot.BoxComponent{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: boxColor,

								PaddingAll: "8px", // ลด Padding
								Contents: []linebot.FlexComponent{
									&linebot.TextComponent{
										Type:   "text",
										Text:   row1,
										Weight: "bold",
										Size:   "lg", // ลดจาก xxl → lg
										Color:  "#FFFFFF",
										Align:  "center",
									},
								},
							},
						}, CornerRadius: "xxl",
						BorderColor: "#FFFFFF",
						// BorderWidth: "2px",
					},
				},
			},
			Body: &linebot.BoxComponent{
				Type:   "box",
				Layout: "vertical",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:   "text",
						Text:   row2,
						Weight: "bold",
						Size:   "sm", // ลดจาก md → sm
						Align:  "center",
						Color:  row2Color,
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   command,
						Weight: "bold",
						Size:   "sm", // ลดจาก md → sm
						Align:  "center",
						Color:  commandColor,
					},
				}, Height: showFullOpen, PaddingAll: "0px",
				CornerRadius: "sm",
				BorderColor:  "#FFFFFF",
			},
		},
	)

	return flexMessage
}
func bet(userID string, command2 string, balance int64, userName string) string {
	var min1, max1, high, value2, value3 int64
	var redSubB1, blueSubJ1 int64
	var maxProfit, maxLost int64
	var com error
	var bNew, jNew int64
	// var totalMaxLost, totalMaxProfit float64
	round, subRound, state, redRate, blueRate, redOpen, blueOpen, command, min, max, win, com = GetLocalVar()
	ctx := context.Background()
	if com != nil {
		log.Println(com)
	}

	// Fetch required values from the database
	// Query to get min, max, high, and round values for the user
	query := `SELECT local_min, local_max, local_max, local_round, local_sub, local_win 
              FROM environment 
              LIMIT 1`
	err := db.QueryRowContext(ctx, query).Scan(&min1, &max1, &high, &round, &subRound, &win)
	if err != nil {
		log.Println(err)
	}

	// Query to fetch max lost and max profit for red and blue teams
	query2 := `
		SELECT 
			'red' AS team, 
			COALESCE(SUM(b1), 0) AS redMaxLost, 
			COALESCE(SUM(maxlost), 0) AS totalMaxLost, 
			COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
		FROM 
			playinglog 
		WHERE 
			  ID = ? AND round = ? AND b1 > 0 AND Game_play!=? AND  Game_play!=?
		UNION ALL
		SELECT 
			'blue' AS team, 
			COALESCE(SUM(j1), 0) AS blueMaxLost, 
			COALESCE(SUM(maxlost), 0) AS totalMaxLost, 
			COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
		FROM 
			playinglog 
		WHERE 
			ID = ? AND round = ? AND j1 > 0 AND Game_play NOT IN (?, ?);
	`
	fmt.Println("Local Min:")
	rows, err := db.QueryContext(ctx, query2, userID, round, "END", "AFTER", userID, round, "END", "AFTER")
	if err != nil {
		fmt.Println("Local Min:")
		log.Println(err)
	}
	defer rows.Close()
	fmt.Println("Local Min:")
	// Retrieve the values for red and blue team

	for rows.Next() {
		var team string
		var sumLost int
		var maxLost float64
		var maxProfit float64 // Using float64 for maxProfit to handle decimal values

		// Assuming 'rows' is the result of a query, we scan the values accordingly
		if err := rows.Scan(&team, &sumLost, &maxLost, &maxProfit); err != nil {
			log.Println(err)
		}

		// After scanning, you can use `sumLost` and `maxLost` as integers and `maxProfit` as a float64

		if err := rows.Scan(&team, &sumLost, &maxLost, &maxProfit); err != nil {
			log.Println(err)
		}
		if team == "red" {
			// Assign the sum of b1 (redSubB1) and maxlost/maxprofit
			redSubB1 = int64(sumLost)       // This is the sum of `b1` for red team
			redMaxLost = int64(maxLost)     // Max lost for red team
			redMaxProfit = int64(maxProfit) // Convert maxProfit (float64) to int for red team
		} else {
			// Assign the sum of j1 (blueSubJ1) and maxlost/maxprofit
			blueSubJ1 = int64(sumLost)       // This is the sum of `j1` for blue team
			blueMaxLost = int64(maxLost)     // Max lost for blue team
			blueMaxProfit = int64(maxProfit) // Convert maxProfit (float64) to int for blue team
		}

	}
	query3 := `
	SELECT 
		'red' AS team, 
		COALESCE(SUM(b1), 0) AS sumLost
	FROM 
		playinglog 
	WHERE 
		ID = ? AND sub = ? AND round = ? AND b1 > 0 AND Game_play NOT IN (?, ?)
	UNION ALL
	SELECT 
		'blue' AS team, 
		COALESCE(SUM(j1), 0) AS sumLost
	FROM 
		playinglog 
	WHERE 
		ID = ? AND sub = ? AND round = ? AND j1 > 0 AND Game_play NOT IN (?, ?);
`

	rows3, err3 := db.QueryContext(ctx, query3, userID, subRound, round, "END", "AFTER", userID, subRound, round, "END", "AFTER")
	if err3 != nil {
		fmt.Println("Error executing query3:", err3)

	}
	defer rows3.Close()

	// Variables for red and blue team sums

	// Retrieve the values for red and blue teams
	for rows3.Next() {
		var team string
		var sumLost int64

		// Scan the row into variables
		if err := rows3.Scan(&team, &sumLost); err != nil {
			log.Println("Error scanning row:", err)
		}

		if team == "red" {
			bNew = sumLost // Assign the sum of b1 (red team)
		} else if team == "blue" {
			jNew = sumLost // Assign the sum of j1 (blue team)
		}
	}

	// Check for errors after iterating through rows
	if err := rows3.Err(); err != nil {
		log.Println("Error iterating rows:", err)
	}

	// Print or use the results
	fmt.Printf("Red Team Sum: %d, Blue Team Sum: %d\n", bNew, jNew)

	fmt.Println("Local Min:")
	// Now process the betting logic
	pairs := command2
	suffix := strings.TrimPrefix(pairs, string(pairs[0:3]))
	key := string(pairs[0:3])
	fmt.Println("Key:", key)
	fmt.Println("pair:", pairs, suffix)
	value, err := strconv.ParseInt(suffix, 10, 64)
	var limitBreak = 0
	var limitBreak2 = 0
	if soi == 0 {
		if key == "ด" {
			if value > high {
				value2 = high // ปรับ value ไม่ให้เกิน high
			} else {
				value2 = value
			}

			if bNew+value2 > high*3 {
				value2 = high*3 - bNew // ปรับ value2 ไม่ให้เกิน high*3
				value = value2
				limitBreak2 = 1
			} else {
				value2 = value
			}

			// กรณีไม่สามารถรับเพิ่มได้เลย
			if value2 <= 0 {
				reply = fmt.Sprintf("(รับสูงสุด %v ฿ ต่อคน)", formatWithCommas(high*3))
				return reply
			}
		}

		if key == "ง" {
			if value > high {
				value3 = high
			} else {
				value3 = value
			}

			if jNew+value3 > high*3 {
				value3 = high*3 - jNew
				value = value3
				limitBreak2 = 1
			} else {
				value3 = value
			}

			if value3 <= 0 {
				reply = fmt.Sprintf("(รับสูงสุด %v ฿ ต่อคน)", formatWithCommas(high*3))
				return reply
			}
		}

		// ล้างยอดชั่วคราว (หากจำเป็น)
		bNew, jNew = 0, 0
	}

	// high = high * 3

	if err != nil {
		log.Println(err)
	}

	if key == "ด" { // Red team
		balance = balance - redMaxLost + blueMaxProfit

		if win == 1 { // Red is underdog
			redMaxLost = int64(value)
			redMaxProfit = int64(math.Round(float64(value) * redRate))
		} else {
			redMaxProfit = int64(value)
			redMaxLost = int64(math.Round(float64(value) * redRate))
		}

		if bNew+value > high {

			scaleFactorRed := float64((high - bNew)) / float64(value)
			value = int64(math.Round(float64(value) * scaleFactorRed))
			redMaxLost = int64(math.Round(float64(redMaxLost) * scaleFactorRed))
			redMaxProfit = int64(math.Round(float64(redMaxProfit) * scaleFactorRed))
			maxLost = redMaxLost
			maxProfit = redMaxProfit
			redSubB1 = value
			blueSubJ1 = 0
			limitBreak = 1

		}

		if redMaxLost > balance && balance >= 1 && value >= 1 {
			if win == 1 {
				value = balance
				redMaxLost = value
				redMaxProfit = int64(math.Round(float64(value) * (float64(redRate))))
			} else {
				value = int64(math.Round(float64(value) * (float64(balance) / float64(redMaxLost))))
				redMaxProfit = value
				redMaxLost = int64(math.Round(float64(value) * (float64(redRate))))
			}
			maxLost = redMaxLost
			maxProfit = redMaxProfit
			redSubB1 = value
			blueSubJ1 = 0

			if value >= 1 {
				InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
				reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %v ฿\n(ได้เสียตามทุน)", key, formatWithCommas(value))
				// reply = ""
			} else {
				reply = "❌ยอดไม่พอเล่น"
				// reply = ""
			}
			return reply
		}
		fmt.Println("KeyR:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit, max1)
		reply = fmt.Sprintf("ฝั่ง %s %d\n", key, value)
		maxLost = redMaxLost
		maxProfit = redMaxProfit
		redSubB1 = value
		blueSubJ1 = 0
	} else {
		balance = balance - blueMaxLost + redMaxProfit
		if win == -1 {
			blueMaxProfit = int64(math.Round(float64(value) * (float64(blueRate))))
			blueMaxLost = value
		} else {
			blueMaxLost = int64(math.Round(float64(value) * (float64(blueRate))))
			blueMaxProfit = value
		}

		if jNew+value > high {
			scaleFactorBlue := (float64(high) - float64(jNew)) / float64(value)
			value = int64(math.Round(float64(value) * scaleFactorBlue))
			blueMaxLost = int64(math.Round(float64(blueMaxLost) * scaleFactorBlue))
			blueMaxProfit = int64(math.Round(float64(blueMaxProfit) * scaleFactorBlue))
			fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
			maxLost = blueMaxLost
			maxProfit = blueMaxProfit
			blueSubJ1 = value
			redSubB1 = 0
			limitBreak = 1

		}

		if blueMaxLost > balance && balance >= 1 && value >= 1 {
			if win == -1 {
				value = int64(math.Round(float64(value) * (float64(balance) / float64(blueMaxLost))))
				blueMaxProfit = int64(math.Round(float64(value) * (float64(blueRate))))
				blueMaxLost = value
			} else {
				value = int64(math.Round(float64(value) * (float64(balance) / float64(blueMaxLost))))
				blueMaxLost = int64(math.Round(float64(value) * (float64(blueRate))))
				blueMaxProfit = value
			}
			fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
			maxLost = blueMaxLost
			maxProfit = blueMaxProfit
			blueSubJ1 = value
			redSubB1 = 0
			if value >= 1 {
				InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
				reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %v ฿\n(ติดเท่าทุน)", key, formatWithCommas(value))
				// reply = ""
			} else {
				reply = "❌ยอดไม่พอเล่น"
				// reply = ""
			}
			return reply

		}
		fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
		maxLost = blueMaxLost
		maxProfit = blueMaxProfit
		reply = fmt.Sprintf("ฝั่ง %s %d\n", key, value)
		blueSubJ1 = value
		redSubB1 = 0
	}
	fmt.Println("Key:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
	if value < 1 && balance < 1 {
		reply = "💸ยอดเงินไม่พอ💸"
		// reply = ""
		return reply
	}

	if value < 1 {
		reply = fmt.Sprintf("(รับสูงสุด %v ฿ ต่อคน)", formatWithCommas(high))
		// reply = ""
		return reply
	}

	if balance <= 0 {
		reply = "💸 ยอดเงินไม่พอ 💸"
		// reply = ""
		return reply
	}
	if limitBreak == 1 && value >= 1 {
		InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
		reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %v ฿\n(ไม่เกิน %v ฿ ต่อไม้)", key, formatWithCommas(value), formatWithCommas(high))
		// reply = ""
		return reply
	} else if limitBreak2 == 1 && value >= 1 {
		InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
		reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %v ฿\n(สูงสุดมุมละ %v ฿)", key, formatWithCommas(value), formatWithCommas(high*3))
		// reply = ""
		return reply
	}
	InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)

	return ""
}
func InsertBet(userID string, names string, redSubB1 int64, round int, maxLost int64, blueSubJ1 int64, maxProfit int64, subRound int, win int, redRate float64, blueRate float64) error {
	InitDB() // Ensure the database connection is initialized
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// SQL query to insert the data into 'playinglog'
	query := `
	INSERT INTO playinglog 
	(ID, Name, b1, round, advanced_pay, j1, maxprofit, maxlost, sub, win, red_rate, blue_rate, Time2)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
`

	// Preparing the values to be inserted
	// var advancedPay int
	// if redSubB1 > 0 {
	// 	advancedPay = redMaxLost
	// } else if blueSubJ1 > 0 {
	// 	advancedPay = blueMaxLost
	// } // You can adjust the logic as per your needs
	// ดึงชื่อจาก user_data ตาม userID
	var nameFromUserData string
	err := db.QueryRowContext(ctx, "SELECT name FROM user_data WHERE ID = ?", userID).Scan(&nameFromUserData)
	if err != nil {
		return fmt.Errorf("error fetching name from user_data: %v", err)
	}

	// เตรียมค่าที่จะใช้ INSERT
	advancedPay := maxLost

	// ทำการ INSERT เข้า playinglog โดยใช้ชื่อที่ดึงมา
	_, err = db.ExecContext(ctx, query, userID, nameFromUserData, redSubB1, round, advancedPay, blueSubJ1, maxProfit, maxLost, subRound, win, redRate, blueRate)
	if err != nil {
		return fmt.Errorf("error inserting bet: %v", err)
	}

	return nil
}
func betOver(userID string, command2 string, balance int64, userName string, rRate float64, bRate float64, wWin int) string {
	var min1, max1, high int64
	var redSubB1, blueSubJ1 int64
	var maxProfit, maxLost int64
	var com error
	var bNew, jNew int64
	// var totalMaxLost, totalMaxProfit float64
	round, subRound, state, redRate, blueRate, redOpen, blueOpen, command, min, max, win, com = GetLocalVar()

	ctx := context.Background()
	if com != nil {
		log.Println(com)
	}

	// Fetch required values from the database
	// Query to get min, max, high, and round values for the user
	query := `SELECT local_min, local_max, local_max, local_round, local_sub, local_win 
              FROM environment 
              LIMIT 1`
	err := db.QueryRowContext(ctx, query).Scan(&min1, &max1, &high, &round, &subRound, &win)
	if err != nil {
		log.Println(err)
	}
	blueRate = bRate
	redRate = rRate
	redOpen = true
	blueOpen = true
	win = wWin
	min = 1
	high = 1000000
	max = 1000000
	subRound = 1
	// Query to fetch max lost and max profit for red and blue teams
	query2 := `
		SELECT 
			'red' AS team, 
			COALESCE(SUM(b1), 0) AS redMaxLost, 
			COALESCE(SUM(maxlost), 0) AS totalMaxLost, 
			COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
		FROM 
			playinglog 
		WHERE 
			  ID = ? AND round = ? AND b1 > 0 AND Game_play!=? AND  Game_play!=?
		UNION ALL
		SELECT 
			'blue' AS team, 
			COALESCE(SUM(j1), 0) AS blueMaxLost, 
			COALESCE(SUM(maxlost), 0) AS totalMaxLost, 
			COALESCE(SUM(maxprofit), 0) AS totalMaxProfit 
		FROM 
			playinglog 
		WHERE 
			ID = ? AND round = ? AND j1 > 0 AND Game_play NOT IN (?, ?);
	`
	fmt.Println("Local Min:")
	rows, err := db.QueryContext(ctx, query2, userID, round, "END", "AFTER", userID, round, "END", "AFTER")
	if err != nil {
		fmt.Println("Local Min:")
		log.Println(err)
	}
	defer rows.Close()
	fmt.Println("Local Min:")
	// Retrieve the values for red and blue team
	for rows.Next() {
		var team string
		var sumLost int
		var maxLost float64
		var maxProfit float64 // Using float64 for maxProfit to handle decimal values

		// Assuming 'rows' is the result of a query, we scan the values accordingly
		if err := rows.Scan(&team, &sumLost, &maxLost, &maxProfit); err != nil {
			log.Println(err)
		}

		// After scanning, you can use `sumLost` and `maxLost` as integers and `maxProfit` as a float64

		if err := rows.Scan(&team, &sumLost, &maxLost, &maxProfit); err != nil {
			log.Println(err)
		}
		if team == "red" {
			// Assign the sum of b1 (redSubB1) and maxlost/maxprofit
			redSubB1 = int64(sumLost)       // This is the sum of `b1` for red team
			redMaxLost = int64(maxLost)     // Max lost for red team
			redMaxProfit = int64(maxProfit) // Convert maxProfit (float64) to int for red team
		} else {
			// Assign the sum of j1 (blueSubJ1) and maxlost/maxprofit
			blueSubJ1 = int64(sumLost)       // This is the sum of `j1` for blue team
			blueMaxLost = int64(maxLost)     // Max lost for blue team
			blueMaxProfit = int64(maxProfit) // Convert maxProfit (float64) to int for blue team
		}

	}
	query3 := `
	SELECT 
		'red' AS team, 
		COALESCE(SUM(b1), 0) AS sumLost
	FROM 
		playinglog 
	WHERE 
		ID = ? AND sub = ? AND round = ? AND b1 > 0 AND Game_play NOT IN (?, ?)
	UNION ALL
	SELECT 
		'blue' AS team, 
		COALESCE(SUM(j1), 0) AS sumLost
	FROM 
		playinglog 
	WHERE 
		ID = ? AND sub = ? AND round = ? AND j1 > 0 AND Game_play NOT IN (?, ?);
`

	rows3, err3 := db.QueryContext(ctx, query3, userID, subRound, round, "END", "AFTER", userID, subRound, round, "END", "AFTER")
	if err3 != nil {
		fmt.Println("Error executing query3:", err3)

	}
	defer rows3.Close()

	// Variables for red and blue team sums

	// Retrieve the values for red and blue teams
	for rows3.Next() {
		var team string
		var sumLost int64

		// Scan the row into variables
		if err := rows3.Scan(&team, &sumLost); err != nil {
			log.Println("Error scanning row:", err)
		}

		if team == "red" {
			bNew = sumLost // Assign the sum of b1 (red team)
		} else if team == "blue" {
			jNew = sumLost // Assign the sum of j1 (blue team)
		}
	}

	// Check for errors after iterating through rows
	if err := rows3.Err(); err != nil {
		log.Println("Error iterating rows:", err)
	}

	// Print or use the results
	fmt.Printf("Red Team Sum: %d, Blue Team Sum: %d\n", bNew, jNew)

	fmt.Println("Local Min:")
	// Now process the betting logic
	pairs := command2

	suffix := strings.TrimPrefix(pairs, string(pairs[0:3]))
	key := string(pairs[0:3])
	fmt.Println("Key:", key)
	fmt.Println("pair:", pairs, suffix)
	value, err := strconv.ParseInt(suffix, 10, 64)
	if err != nil {
		log.Println(err)
	}
	var limitBreak = 0
	if key == "ด" { // Red team
		balance = balance - redMaxLost + blueMaxProfit

		if win == 1 { // Red is underdog
			redMaxLost = int64(value)
			redMaxProfit = int64(float64(value) * redRate)
		} else {
			redMaxProfit = int64(value)
			redMaxLost = int64(float64(value) * redRate)
		}

		if bNew+value > high {

			scaleFactorRed := float64((high - bNew)) / float64(value)
			value = int64(float64(value) * scaleFactorRed)
			redMaxLost = int64(float64(redMaxLost) * scaleFactorRed)
			redMaxProfit = int64(float64(redMaxProfit) * scaleFactorRed)
			maxLost = redMaxLost
			maxProfit = redMaxProfit
			redSubB1 = value
			blueSubJ1 = 0
			limitBreak = 1

		}

		if redMaxLost > balance && balance >= 1 && value > 1 {
			if win == 1 {
				value = balance
				redMaxLost = value
				redMaxProfit = int64(float64(value) * (float64(redRate)))
			} else {
				value = int64(float64(value) * (float64(balance) / float64(redMaxLost)))
				redMaxProfit = value
				redMaxLost = int64(float64(value) * (float64(redRate)))
			}
			maxLost = redMaxLost
			maxProfit = redMaxProfit
			redSubB1 = value
			blueSubJ1 = 0
			if value >= 1 {
				InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
				reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %d ฿\n(ติดเท่าทุน)", key, value)
			} else {
				reply = "❌ยอดไม่พอเล่น"
			}

			return reply
		}
		fmt.Println("KeyR:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit, max1)
		reply = fmt.Sprintf("ฝั่ง %s %d\n", key, value)
		maxLost = redMaxLost
		maxProfit = redMaxProfit
		redSubB1 = value
		blueSubJ1 = 0
	} else {
		balance = balance - blueMaxLost + redMaxProfit
		if win == -1 {
			blueMaxProfit = int64(float64(value) * (float64(blueRate)))
			blueMaxLost = value
		} else {
			blueMaxLost = int64(float64(value) * (float64(blueRate)))
			blueMaxProfit = value
		}

		if jNew+value > high {
			scaleFactorBlue := (float64(high) - float64(jNew)) / float64(value)
			value = int64(float64(value) * scaleFactorBlue)
			blueMaxLost = int64(float64(blueMaxLost) * scaleFactorBlue)
			blueMaxProfit = int64(float64(blueMaxProfit) * scaleFactorBlue)
			fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
			maxLost = blueMaxLost
			maxProfit = blueMaxProfit
			blueSubJ1 = value
			redSubB1 = 0
			limitBreak = 1

		}

		if blueMaxLost > balance && balance >= 1 {
			if win == -1 {
				value = int64(float64(value) * (float64(balance) / float64(blueMaxLost)))
				blueMaxProfit = int64(float64(value) * (float64(blueRate)))
				blueMaxLost = value
			} else {
				value = int64(float64(value) * (float64(balance) / float64(blueMaxLost)))
				blueMaxLost = int64(float64(value) * (float64(blueRate)))
				blueMaxProfit = value
			}
			fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
			maxLost = blueMaxLost
			maxProfit = blueMaxProfit
			blueSubJ1 = value
			redSubB1 = 0
			if value >= 1 {
				InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
				reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %d ฿\n(ติดเท่าทุน)", key, value)
			} else {
				reply = "❌ยอดไม่พอเล่น"
			}
			return reply

		}
		fmt.Println("KeyB:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
		maxLost = blueMaxLost
		maxProfit = blueMaxProfit
		reply = fmt.Sprintf("ฝั่ง %s %d\n", key, value)
		blueSubJ1 = value
		redSubB1 = 0
	}
	fmt.Println("Key:", maxLost, maxProfit, blueSubJ1, redSubB1, blueMaxLost, blueMaxProfit, redMaxLost, redMaxProfit)
	if value <= 1 {
		reply = "ทุนไม่พอ กรุณาฝากเงิน"
		return reply
	}

	if value < 1 {
		reply = "ลงเต็มจำนวนแล้ว"
		return reply
	}

	if balance <= 0 {
		reply = "ยอดคงเหลือไม่พอ"
		return reply
	}
	if limitBreak == 1 {
		InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)
		reply = fmt.Sprintf("✅️ปรับยอดแทง✅️ %s %d ฿\n(ไม่เกิน %d ฿ ต่อไม้)", key, value, high)
		return reply
	}
	InsertBet(userID, userName, redSubB1, round, maxLost, blueSubJ1, maxProfit, subRound, win, redRate, blueRate)

	return fmt.Sprintf("%d", value)
}

type UserSum struct {
	Sum        float64
	Sum2       float64
	BossCredit float64
	TempCredit float64
	Name       string
	UserNumber string
	Cr         float64
}

func setState(state int) {
	updateQuery := fmt.Sprintf(`
        UPDATE environment 
        SET 
            local_state = %d
        WHERE 1
    `, state)

	// Print or execute the SQL query
	ctx := context.Background()
	fmt.Println(updateQuery)
	ExecuteQuery(ctx, updateQuery)
}
func Summarize2(command string, round int) [][]string {
	localRound, localSub, localState, localRedRate, localBlueRate, _,
		_, localCommand, localMin, localMax, localWin, err := GetLocalVar()

	if err != nil {
		log.Printf("Error getting local variables: %v", err)
		return nil
	}

	log.Printf("Local Variables: Round=%d, Sub=%d, State=%d, RedRate=%.2f, BlueRate=%.2f, Command=%s, Min=%d, Max=%d, Win=%d",
		localRound, localSub, localState, localRedRate, localBlueRate, localCommand, localMin, localMax, localWin)

	query := `
		SELECT pl.ID, pl.b1, pl.j1, pl.maxlost, pl.maxprofit, pl.Name, 
		       ud.Number, ud.Credit, ud.Credit2, ud.multiply
		FROM playinglog pl 
		JOIN user_data ud ON pl.ID = ud.ID 
		WHERE pl.round = ? AND pl.Game_play != ? AND pl.Game_play != ?`

	InitDB()
	rows, err := db.Query(query, localRound, "END", "AFTER")
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil
	}
	defer rows.Close()

	userSums := make(map[string]*UserSum)
	userNumbers := make(map[string]string)
	userMultis := make(map[string]float64)

	for rows.Next() {
		var (
			userID     string
			b1, j1     sql.NullFloat64
			maxLost    sql.NullFloat64
			maxProfit  sql.NullFloat64
			name       string
			userNumber string
			credit     sql.NullFloat64
			credit2    sql.NullFloat64
			multiply   sql.NullFloat64
		)

		err := rows.Scan(&userID, &b1, &j1, &maxLost, &maxProfit, &name, &userNumber, &credit, &credit2, &multiply)
		if err != nil {
			log.Printf("Error scanning row for user %s: %v", userID, err)
			continue
		}

		userNumbers[userID] = userNumber

		multi := 1.0
		if multiply.Valid && multiply.Float64 >= 1.0 {
			multi = multiply.Float64
		}
		userMultis[userID] = multi

		var sum float64
		if command == "R" {
			if b1.Valid && b1.Float64 > 0 {
				sum += maxProfit.Float64
			}
			if j1.Valid && j1.Float64 > 0 {
				sum -= maxLost.Float64
			}
		}
		if command == "B" {
			if b1.Valid && b1.Float64 > 0 {
				sum -= maxLost.Float64
			}
			if j1.Valid && j1.Float64 > 0 {
				sum += maxProfit.Float64
			}
		}

		if _, exists := userSums[userID]; !exists {
			userSums[userID] = &UserSum{
				Name:       fmt.Sprintf("%s.%s", userNumber, name),
				Cr:         credit.Float64,
				Sum:        0,
				BossCredit: credit2.Float64,
				TempCredit: credit.Float64,
			}
		}

		userData := userSums[userID]
		userData.Sum += sum
		userData.BossCredit = credit2.Float64
		userData.Cr += sum

		log.Printf("User %s updated: Sum=%.2f, BossCredit=%.2f, TempCredit=%.2f, Credit=%.2f, Multiply=%.2f",
			userID, userData.Sum, userData.BossCredit, userData.TempCredit, userData.Cr, multi)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error processing rows: %v", err)
		return nil
	}

	type sortableUser struct {
		ID        string
		Remain    int
		NumberInt int
		Data      *UserSum
	}

	var sortedUsers []sortableUser
	for id, data := range userSums {
		numberInt, _ := strconv.Atoi(userNumbers[id])
		remain := int(data.Cr) - int(data.BossCredit)

		sortedUsers = append(sortedUsers, sortableUser{
			ID:        id,
			Remain:    remain,
			NumberInt: numberInt,
			Data:      data,
		})
	}

	sort.Slice(sortedUsers, func(i, j int) bool {
		if sortedUsers[i].NumberInt != sortedUsers[j].NumberInt {
			return sortedUsers[i].NumberInt < sortedUsers[j].NumberInt
		}
		return sortedUsers[i].NumberInt < sortedUsers[j].NumberInt
	})

	var messageBuilder strings.Builder
	messageBuilder.WriteString("ผู้เล่น//,// ได้เสีย  คงเหลือ\n")

	for _, user := range sortedUsers {
		data := user.Data
		multiplier := userMultis[user.ID]

		adjustedSum := data.Sum * multiplier
		newCredit := data.TempCredit + adjustedSum

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := ExecuteQuery(ctx, "UPDATE user_data SET Credit = ? WHERE ID = ?", newCredit, user.ID)
		cancel()
		if err != nil {
			log.Printf("Error updating user %s credit: %v", user.ID, err)
			continue
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		err2 := ExecuteQuery(ctx2,
			"UPDATE playinglog SET balance = ?, adminSum = ? WHERE ID = ? AND round = ? AND Status = ?",
			adjustedSum, newCredit, user.ID, localRound, "")
		cancel2()
		if err2 != nil {
			log.Printf("Error updating playinglog for user %s: %v", user.ID, err2)
			continue
		}

		messageBuilder.WriteString(fmt.Sprintf("%s//,//%d %d  คงเหลือ %d\n",
			data.Name,
			int(data.TempCredit)-int(data.BossCredit),
			int(adjustedSum),
			int(newCredit)-int(data.BossCredit)))
	}

	if messageBuilder.Len() == 0 {
		return [][]string{}
	}
	return splitAndFormatMessage(messageBuilder.String())
}
func Summarize2p(command string, round int) [][]string {
	localRound, localSub, localState, localRedRate, localBlueRate, _,
		_, localCommand, localMin, localMax, localWin, err := GetLocalVar()

	if err != nil {
		log.Printf("Error getting local variables: %v", err)
		return nil
	}
	localRound -= 1

	log.Printf("Local Variables: Round=%d, Sub=%d, State=%d, RedRate=%.2f, BlueRate=%.2f, Command=%s, Min=%d, Max=%d, Win=%d",
		localRound, localSub, localState, localRedRate, localBlueRate, localCommand, localMin, localMax, localWin)

	query := `
		SELECT pl.ID, pl.b1, pl.j1, pl.maxlost, pl.maxprofit, pl.Name, ud.Number, ud.Credit , ud.Credit2
		FROM playinglog pl 
		JOIN user_data ud ON pl.ID = ud.ID 
		WHERE pl.round = ? AND pl.Game_play != ? AND pl.Game_play != ?`

	InitDB()
	rows, err := db.Query(query, localRound, "END", "AFTER")
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil
	}
	defer rows.Close()

	userSums := make(map[string]*UserSum)
	userNumbers := make(map[string]string)

	for rows.Next() {
		var (
			userID     string
			b1, j1     sql.NullFloat64
			maxLost    sql.NullFloat64
			maxProfit  sql.NullFloat64
			name       string
			userNumber string
			credit     sql.NullFloat64
			credit2    sql.NullFloat64
		)

		err := rows.Scan(&userID, &b1, &j1, &maxLost, &maxProfit, &name, &userNumber, &credit, &credit2)
		if err != nil {
			log.Printf("Error scanning row for user %s: %v", userID, err)
			continue
		}

		userNumbers[userID] = userNumber

		var sum float64
		if command == "R" {
			if b1.Valid && b1.Float64 > 0 {
				sum += maxProfit.Float64
			}
			if j1.Valid && j1.Float64 > 0 {
				sum -= maxLost.Float64
			}
		}
		if command == "B" {
			if b1.Valid && b1.Float64 > 0 {
				sum -= maxLost.Float64
			}
			if j1.Valid && j1.Float64 > 0 {
				sum += maxProfit.Float64
			}
		}
		if command == "S" {
			// ไม่เปลี่ยนค่า sum
		}

		if _, exists := userSums[userID]; !exists {
			userSums[userID] = &UserSum{
				Name:       fmt.Sprintf("%s.%s", userNumber, name),
				Cr:         credit.Float64,
				Sum:        0,
				BossCredit: credit2.Float64,
				TempCredit: credit.Float64,
			}
		}

		userData := userSums[userID]
		userData.Sum += sum
		userData.BossCredit = credit2.Float64
		userData.Cr += sum * 0
		userData.TempCredit -= sum

		log.Printf("User %s updated: Sum=%.2f, BossCredit=%.2f, TempCredit=%.2f, Credit=%.2f",
			userID, userData.Sum, userData.BossCredit, userData.TempCredit, userData.Cr)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error processing rows: %v", err)
		return nil
	}

	// ---- เรียงลำดับก่อนแสดงผล ----
	type sortableUser struct {
		ID        string
		Remain    int
		NumberInt int
		Data      *UserSum
	}

	var sortedUsers []sortableUser
	for id, data := range userSums {
		numberInt, _ := strconv.Atoi(userNumbers[id]) // ถ้าแปลงไม่ได้ = 0
		remain := int(data.Cr) - int(data.BossCredit)

		sortedUsers = append(sortedUsers, sortableUser{
			ID:        id,
			Remain:    remain,
			NumberInt: numberInt,
			Data:      data,
		})
	}

	sort.Slice(sortedUsers, func(i, j int) bool {
		if sortedUsers[i].NumberInt != sortedUsers[j].NumberInt {
			return sortedUsers[i].NumberInt < sortedUsers[j].NumberInt
		}
		return sortedUsers[i].NumberInt < sortedUsers[j].NumberInt
	})

	// ---- สร้างข้อความตอบกลับ ----
	var messageBuilder strings.Builder
	messageBuilder.WriteString("ผู้เล่น//,// ได้เสีย  คงเหลือ\n")

	for _, user := range sortedUsers {
		data := user.Data

		// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// err := ExecuteQuery(ctx, "UPDATE user_data SET Credit = ? WHERE ID = ?", data.Cr, user.ID)
		// cancel()
		// if err != nil {
		// 	log.Printf("Error updating user %s credit: %v", user.ID, err)
		// 	continue
		// }

		// ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		// err2 := ExecuteQuery(ctx2, "UPDATE playinglog SET balance = ?, adminSum = ? WHERE ID = ? AND round = ? AND Status = ?", data.Sum, data.Cr, user.ID, localRound, "")
		// cancel2()
		// if err2 != nil {
		// 	log.Printf("Error updating playinglog for user %s: %v", user.ID, err2)
		// 	continue
		// }

		messageBuilder.WriteString(fmt.Sprintf("%s//,//%d %d  คงเหลือ %d\n",
			data.Name,
			int(data.TempCredit)-int(data.BossCredit),
			int(data.Sum),
			int(data.Cr)-int(data.BossCredit)))
	}

	if messageBuilder.Len() == 0 {
		return [][]string{}
	}
	return splitAndFormatMessage(messageBuilder.String())
}
func summarize4(command string, round int) [][]string {
	dsn := "duckcom_fulloption2:duckcom_fulloption2@tcp(203.170.129.1:3306)/duckcom_fulloption2"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("Error opening database: %v", err)
	}
	defer db.Close()

	// Query to fetch the relevant data
	query := `
		SELECT ID, Name, Credit, Credit2, Number 
		FROM user_data 
		WHERE (Credit - Credit2) != ?
		ORDER BY (Credit2 - Credit) DESC
	`
	rows, err := db.Query(query, 0)
	if err != nil {
		log.Println("Error executing query: %v", err)
	}
	defer rows.Close()

	type UserSum struct {
		UserID     string
		Name       string
		Sum        float64
		BossCredit float64
		TempCredit float64
		Number     int64
	}

	// Collect user sums
	var userSums []UserSum

	for rows.Next() {
		var (
			userID  string
			name    string
			credit  sql.NullFloat64
			credit2 sql.NullFloat64
			number  sql.NullInt64
		)

		err := rows.Scan(&userID, &name, &credit, &credit2, &number)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		if !credit.Valid || !credit2.Valid || !number.Valid {
			log.Printf("Skipping row with invalid data: userID=%s", userID)
			continue
		}

		sum := credit.Float64 - credit2.Float64

		userSums = append(userSums, UserSum{
			UserID:     userID,
			Name:       name,
			Sum:        sum,
			BossCredit: 0, // Placeholder, update logic if needed
			TempCredit: 0, // Placeholder, update logic if needed
			Number:     number.Int64,
		})
	}

	if err := rows.Err(); err != nil {
		log.Println("Error processing rows: %v", err)
	}

	// Sort user sums by the Sum field in descending order
	sort.Slice(userSums, func(i, j int) bool {
		return userSums[i].Sum > userSums[j].Sum
	})

	// Build the final message
	var message strings.Builder
	for _, user := range userSums {
		// Example of additional processing if needed
		cs := getBalance(user.UserID) - getBalance2(user.UserID)
		dp := getDeposit(user.UserID)
		// for i := 0; i < 15; i++ {
		message.WriteString(fmt.Sprintf("%d.%s//,//%d %d %d\n",
			user.Number, user.Name, dp, cs-dp, cs))
		// }

	}

	if message.Len() == 0 {
		return flexPost("----")
	}

	return flexPost(message.String())
}

func flexPost(message string) [][]string {
	lines := strings.Split(message, "\n")
	var array1, array2 []string

	for _, line := range lines {
		if line != "" {
			parts := strings.Split(line, "//,//")
			if len(parts) == 2 {
				array1 = append(array1, parts[0])
				array2 = append(array2, parts[1])
			}
		}
	}

	// Combine array1 and array2 into a 2D slice
	var combinedArray [][]string
	for i := 0; i < len(array1); i++ {
		combinedArray = append(combinedArray, []string{array1[i], array2[i]})
	}
	return combinedArray
}
func deniedSub(db *sql.DB, round int, sub int, state int) string {
	if state == 2 {
		// Prepare the SQL query to select the names
		query := fmt.Sprintf(`
		SELECT Name FROM playinglog 
		WHERE Game_play != '%s' AND Game_play != '%s' AND round = %d AND sub = %d
		`, "END", "AFTER", round, sub)

		// Create a context for executing the query
		ctx := context.Background()

		// Execute the SELECT query
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			fmt.Println("Error executing query:", err)
			return "Error"
		}
		defer rows.Close()

		// Process the result set
		var advancedPay string
		advancedPay += "การเดิมพันรอบล่าสุดของท่านถูก \"ยกเลิก\"" + "\n"
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				fmt.Println("Error scanning row:", err)
				return "Error"
			}
			advancedPay += "@" + name + "\n"
		}

		// Check if there were no results
		if advancedPay == "การเดิมพันรอบล่าสุดของท่านถูก \"ยกเลิก\""+"\n" {
			advancedPay = "การเดิมพันรอบล่าสุดของท่านถูก \"ยกเลิก\"" + "\n ----------"
		}

		// Prepare and execute the DELETE query
		deleteQuery := fmt.Sprintf(`
		DELETE FROM playinglog 
		WHERE Game_play != '%s' AND round = %d AND sub = %d
		`, "END", round, sub)

		_, err = db.ExecContext(ctx, deleteQuery)
		if err != nil {
			fmt.Println("Error executing delete query:", err)
			return "Error"
		}

		// Return the result
		return advancedPay
	}
	return ""
}

func Summarize(command string, round int) [][]string {
	localRound, localSub, localState, localRedRate, localBlueRate, _,
		_, localCommand, localMin, localMax, localWin, err := GetLocalVar()

	if err != nil {
		log.Printf("Error getting local variables: %v", err)
		return nil
	}

	// Log local variables
	log.Printf("Local Variables: Round=%d, Sub=%d, State=%d, RedRate=%.2f, BlueRate=%.2f, Command=%s, Min=%d, Max=%d, Win=%d",
		localRound, localSub, localState, localRedRate, localBlueRate, localCommand, localMin, localMax, localWin)

	query := `
		SELECT 
			pl.ID, pl.b1, pl.j1, pl.maxlost, pl.maxprofit, 
			pl.Name, ud.Number, ud.Credit 
		FROM 
			playinglog pl 
		JOIN 
			user_data ud ON pl.ID = ud.ID 
		WHERE 
			pl.round = ? AND 
			pl.Game_play != ? AND 
			pl.Game_play != ? AND 
			pl.sub = ?
		ORDER BY 
			ud.Number ASC
	`

	InitDB()
	rows, err := db.Query(query, localRound, "END", "AFTER", localSub)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil
	}
	defer rows.Close()

	// Map to aggregate user data
	userSums := make(map[string]*UserSum)
	var userOrder []string // Slice to preserve the insertion order

	for rows.Next() {
		var (
			userID     string
			b1, j1     sql.NullFloat64
			maxLost    sql.NullFloat64
			maxProfit  sql.NullFloat64
			name       string
			userNumber string
			credit     sql.NullFloat64
		)

		// Scan row values
		err := rows.Scan(&userID, &b1, &j1, &maxLost, &maxProfit, &name, &userNumber, &credit)
		if err != nil {
			log.Printf("Error scanning row for user %s: %v", userID, err)
			continue
		}

		// Calculate sum
		var sum, bSum, jSum float64
		if b1.Valid && b1.Float64 > 0 {
			if command != "" && maxProfit.Valid {
				bSum += b1.Float64
			}
		}
		if j1.Valid && j1.Float64 > 0 {
			if command != "" && maxProfit.Valid {
				jSum += j1.Float64
			}
		}

		// Initialize or update user data in the map
		if _, exists := userSums[userID]; !exists {
			userSums[userID] = &UserSum{
				Name:       fmt.Sprintf("%s.%s", userNumber, name),
				Cr:         credit.Float64,
				Sum:        0,
				BossCredit: 0,
				TempCredit: 0,
			}
			userOrder = append(userOrder, userID) // Store the order of user IDs
		}

		// Aggregate sums
		userData := userSums[userID]
		userData.Sum += sum
		userData.BossCredit += bSum
		userData.TempCredit += jSum
		userData.Cr += sum

		log.Printf("User %s updated: Sum=%.2f, BossCredit=%.2f, TempCredit=%.2f, Credit=%.2f",
			userID, userData.Sum, userData.BossCredit, userData.TempCredit, userData.Cr)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error processing rows: %v", err)
		return nil
	}

	// Prepare and execute updates
	var messageBuilder strings.Builder
	messageBuilder.WriteString("ชื่อ//,// ได้เสีย  คงเหลือ\n")

	// Iterate over the userOrder slice to preserve insertion order
	for _, userID := range userOrder {
		data := userSums[userID]
		messageBuilder.WriteString(fmt.Sprintf("%s//,// %d  %d\n",
			data.Name, int(data.BossCredit), int(data.TempCredit)))
	}

	// Return formatted output
	if messageBuilder.Len() == 0 {
		return [][]string{}
	}
	return splitAndFormatMessage(messageBuilder.String())
}

// Split and format the message
func splitAndFormatMessage(message string) [][]string {
	lines := make([][]string, 0)
	for _, line := range strings.Split(message, "\n") {
		if line != "" {
			parts := strings.Split(line, "//,//")
			lines = append(lines, parts)
		}
		fmt.Println(lines)
	}
	return lines
}

// OPEN and CLOSE
func getDeposit(UserID string) int64 {
	// Define the current date and calculate start and end dates for 2 AM to 2 AM window
	// startTime := time.Now().Truncate(24 * time.Hour).Add(2 * time.Hour) // Set to 2 AM today
	// endTime := startTime.Add(24 * time.Hour)                            // Set to 2 AM next day

	// Format the start and end time into the required format for SQL
	// startDateSQL := startTime.Format("2006-01-02 15:04:05")
	// endDateSQL := endTime.Format("2006-01-02 15:04:05")

	// Define your SQL query
	var idNum int // Declare the variable to store the result

	// Execute the query
	ctx := context.Background()
	err4 := db.QueryRowContext(ctx, `SELECT number FROM user_data WHERE id = ? LIMIT 1`, UserID).Scan(&idNum)
	if err4 != nil {

	}
	query := `
		SELECT 
			
			SUM(CASE WHEN STATE = 'ฝาก' THEN AMOUNT ELSE 0 END) + 
			SUM(CASE WHEN STATE = 'ถอน' THEN AMOUNT ELSE 0 END) AS TotalCredit
		FROM wd
		WHERE  UID = ? AND checks = 0
	
	`

	// Initialize DB Connection
	InitDB()

	// Create a variable to store the result
	var totalCredit sql.NullFloat64

	// Use QueryRow for a single result
	err := db.QueryRow(query, idNum).Scan(&totalCredit)
	if err != nil {
		if err == sql.ErrNoRows {
			// Handle case where no result is found
			log.Printf("No balance found for UserID: %s", UserID)
		} else {
			// Handle other errors
			log.Printf("Error executing query: %v", err)
		}
		return 0 // Return 0 if no balance found or if an error occurs
	}

	// Check if the result is valid (not NULL)
	if totalCredit.Valid {
		// Convert float64 to int64 if valid
		return int64(totalCredit.Float64)
	}

	// If the totalCredit value is NULL, return 0
	log.Printf("TotalCredit is NULL for UserID: %s", UserID)
	return 0
}

func getBalance(UserID string) int64 {
	// Define your SQL query
	query := `SELECT Credit FROM user_data WHERE ID = ?` // Use parameterized query to prevent SQL injection

	// Initialize DB Connection
	InitDB()

	// Create a variable to store the result
	var credit sql.NullFloat64

	// Use QueryRow for a single result
	err := db.QueryRow(query, UserID).Scan(&credit)
	if err != nil {
		if err == sql.ErrNoRows {
			// Handle case where no result is found
			log.Printf("No balance found for UserID: %s", UserID)
		} else {
			// Handle other errors
			log.Printf("Error executing query: %v", err)
		}
		return 0 // Return 0 if no credit found or if an error occurs
	}

	// Check if the result is valid (not NULL)
	if credit.Valid {
		// Convert float64 to int64 if valid
		return int64(credit.Float64)
	}

	// If the credit value is NULL, return 0
	log.Printf("Credit is NULL for UserID: %s", UserID)
	return 0
}

func getBalance2(UserID string) int64 {
	// Define your SQL query
	query := `SELECT Credit2 FROM user_data WHERE ID = ?` // Use parameterized query to prevent SQL injection

	// Initialize DB Connection
	InitDB()

	// Create a variable to store the result
	var credit sql.NullFloat64

	// Use QueryRow for a single result
	err := db.QueryRow(query, UserID).Scan(&credit)
	if err != nil {
		if err == sql.ErrNoRows {
			// Handle case where no result is found
			log.Printf("No balance found for UserID: %s", UserID)
		} else {
			// Handle other errors
			log.Printf("Error executing query: %v", err)
		}
		return 0 // Return 0 if no credit found or if an error occurs
	}

	// Check if the result is valid (not NULL)
	if credit.Valid {
		// Convert float64 to int64 if valid
		return int64(credit.Float64)
	}

	// If the credit value is NULL, return 0
	log.Printf("Credit2 is NULL for UserID: %s", UserID)
	return 0
}

// FlexMessage represents the entire Flex Message with a carousel of bubbles.
// FlexMessage represents a Flex message for sending to the LINE API.
type FlexMessage struct {
	Type     string        `json:"type"`     // The type of message (must be "flex")
	Contents []interface{} `json:"contents"` // The content of the Flex message, which is an array of boxes or bubbles
	AltText  string        `json:"altText,omitempty"`
}

// Box represents a box component in Flex messages.
type Button struct {
	Type   string `json:"type"`
	Action Action `json:"action"`
	Style  string `json:"style"`
	Color  string `json:"color"`
}
type Action struct {
	Type          string `json:"type"`
	Label         string `json:"label,omitempty"`
	ClipboardText string `json:"clipboardText,omitempty"`
	URI           string `json:"uri,omitempty"`
}
type Box struct {
	Type            string        `json:"type"`                      // Type of the box (e.g., "box", "image", "text")
	Layout          string        `json:"layout,omitempty"`          // Layout type for the box (e.g., "horizontal", "vertical")
	Spacing         string        `json:"spacing,omitempty"`         // Spacing between elements (e.g., "sm", "md")
	URL             string        `json:"url,omitempty"`             // URL of an image, if it's an image box
	Text            string        `json:"text,omitempty"`            // Text content for text boxes
	Weight          string        `json:"weight,omitempty"`          // Weight for text (e.g., "bold", "regular")
	Size            string        `json:"size,omitempty"`            // Size of the text (e.g., "sm", "lg", "xs")
	Color           string        `json:"color,omitempty"`           // Color of the text
	Align           string        `json:"align,omitempty"`           // Alignment for text ("center", "start", "end")
	TextDecoration  string        `json:"textDecoration,omitempty"`  // Text decoration (e.g., "underline")
	Height          string        `json:"height,omitempty"`          // Height of the box
	Contents        []interface{} `json:"contents,omitempty"`        // Contains other boxes or elements
	JustifyContent  string        `json:"justifyContent,omitempty"`  // Justify content
	Margin          string        `json:"margin,omitempty"`          // Margin around the box
	Width           string        `json:"width,omitempty"`           // Width of the box
	BackgroundColor string        `json:"backgroundColor,omitempty"` // Background color of the box
	Padding         string        `json:"padding,omitempty"`         // Padding inside the box
	BorderColor     string        `json:"borderColor,omitempty"`     // Simulated border color (for visual purposes)
	BorderWidth     string        `json:"borderWidth,omitempty"`     // Simulated border width
	CornerRadius    string        `json:"cornerRadius,omitempty"`    // Radius for box corners (e.g., "sm", "md", "lg", "10px")
	Wrap            bool          `json:"wrap,omitempty"`            // Whether to wrap text or contents inside the box
	Flex            int           `json:"flex,omitempty"`            // Flex value for layout control
	Action          *Action       `json:"action,omitempty"`          // Action field for URI or other actions
}

// Bubble represents a bubble in the carousel (can include hero, body, footer).
type Bubble struct {
	Type   string `json:"type"`             // The type of bubble (usually "bubble")
	Size   string `json:"size,omitempty"`   // The size of the bubble (e.g., "nano", "giga", "mega")
	Hero   Box    `json:"hero,omitempty"`   // The hero image component
	Body   Box    `json:"body,omitempty"`   // The body section (text and other elements)
	Footer Box    `json:"footer,omitempty"` // Footer section (optional)

}

type Image struct {
	Type       string `json:"type"`
	Url        string `json:"url"`
	Height     string `json:"height,omitempty"`
	AspectMode string `json:"aspectMode,omitempty"`
	OffsetEnd  string `json:"offsetEnd,omitempty"`
	Width      string `json:"width,omitempty"`
	Size       string `json:"size,omitempty"`
}

type Text struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	Size         string `json:"size,omitempty"`
	Align        string `json:"align,omitempty"`
	Color        string `json:"color,omitempty"`
	Weight       string `json:"weight,omitempty"`
	Gravity      string `json:"gravity,omitempty"`
	OffsetBottom string `json:"offsetBottom,omitempty"`
	OffsetStart  string `json:"offsetStart,omitempty"`
}

type Separator struct {
	Type   string `json:"type"`
	Margin string `json:"margin,omitempty"`
}

// Function to parse the input string and extract player data
// func parsePlayerData(input string) ([]map[string]string, error) {
// 	var playerData []map[string]string
// 	// Regular expression to match player data like (6)James JR -11 คงเหลือ 104088
// 	re := regexp.MustCompile(`\[(\d+)\)([a-zA-Z]+) ([a-zA-Z]+) (-?\d+) คงเหลือ (-?\d+)`)
// 	matches := re.FindAllStringSubmatch(input, -1)

// 	for _, match := range matches {
// 		if len(match) == 6 {
// 			player := map[string]string{
// 				"ID":      match[1],
// 				"Name":    fmt.Sprintf("%s %s", match[2], match[3]),
// 				"Result":  match[4],
// 				"Balance": match[5],
// 			}
// 			playerData = append(playerData, player)
// 		}
// 	}

// 	return playerData, nil
// }

// type Box struct {
// 	Type     string        `json:"type"`
// 	Layout   string        `json:"layout,omitempty"`
// 	Text     string        `json:"text,omitempty"`
// 	Weight   string        `json:"weight,omitempty"`
// 	Size     string        `json:"size,omitempty"`
// 	Color    string        `json:"color,omitempty"`
// 	Align    string        `json:"align,omitempty"`
// 	Contents []interface{} `json:"contents,omitempty"`
// }

// // Bubble represents a bubble in the carousel.
// type Bubble struct {
// 	Type   string `json:"type"`
// 	Size   string `json:"size,omitempty"`
// 	Hero   Box    `json:"hero,omitempty"`
// 	Body   Box    `json:"body,omitempty"`
// 	Footer Box    `json:"footer,omitempty"`
// }

// // FlexMessage represents the whole Flex message structure.
//
//	type FlexMessage struct {
//		Type     string        `json:"type"`
//		Contents []interface{} `json:"contents"`
//	}
func extractNumbers(input string) (string, string, error) {
	// Define the regular expression to match numbers, including negative ones
	re := regexp.MustCompile(`-?\d+`)

	// Find all matches of numbers in the string
	matches := re.FindAllString(input, -1)

	if len(matches) < 2 {
		return "", "", fmt.Errorf("expected at least two numbers, but got %d", len(matches))
	}

	// Convert the matched strings to integers
	num1 := matches[0]
	num2 := matches[1]

	// Return the numbers as integers
	return num1, num2, nil
}
func extractNumbers2(input string) (string, string, string, error) {
	// Define the regular expression to match numbers, including negative ones
	re := regexp.MustCompile(`-?\d+`)

	// Find all matches of numbers in the string
	matches := re.FindAllString(input, -1)

	if len(matches) < 2 {
		return "", "", "", fmt.Errorf("expected at least two numbers, but got %d", len(matches))
	}

	// Convert the matched strings to integers
	num1 := matches[0]
	num2 := matches[1]
	num3 := matches[2]

	// Return the numbers as integers
	return num1, num2, num3, nil
}
func getColor(value int) string {
	if value > 0 {
		return "#000000" // Green for positive values
	} else if value < 0 {
		return "#FF0000" // Red for negative values
	}
	return "#000000" // Black for zero
}
func getColor2(value int) string {
	if value > 0 {
		return "#000000" // Green for positive values
	} else if value < 0 {
		return "#FF0000" // Red for negative values
	}
	return "#000000" // Black for zero
}
func getColor22(value int) string {
	if value > 0 {
		return "#ffffff" // Green for positive values
	} else if value < 0 {
		return "#FF0000" // Red for negative values
	}
	return "#ffffff" // Black for zero
}
func GenerateFlexMessageOLD(te [][]string) (*FlexMessage, error) {
	// Initialize variables
	var flexMessages []interface{}
	var contents []interface{}
	rowCount := 0
	maxRows := 100 // Limit the rows to 100

	// Header row (Name, Get, Remaining)
	contents = append(contents, Box{
		Type:   "box",        // This is the container for the horizontal layout
		Layout: "horizontal", // Set the layout to horizontal
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "ชื่อ",
				Weight: "bold",
				Size:   "xs",
				Align:  "start",
			},
			Box{
				Type:   "text",
				Text:   "แดง",
				Weight: "bold",
				Size:   "xs",
				Align:  "end",
				Color:  "#FF0000",
			},
			Box{
				Type:   "text",
				Text:   "น้ำเงิน",
				Weight: "bold",
				Size:   "xs",
				Align:  "end",
				Color:  "#0000FF",
			},
		},
	})
	contents = append(contents, Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			Box{
				Type: "separator",
			},
		},
	})

	// Iterate through the 2D array to create dynamic rows based on input
	for _, row := range te[1:] { // Skip the header row
		if rowCount >= maxRows { // Stop processing if we reach 100 rows
			break
		}
		var formGet, formRemain string
		// If data is empty, replace it with "----"
		name := row[0]
		if name == "" {
			name = "----"
		}

		get, remaining, _ := extractNumbers(row[1])
		if get == "" {
			get = "----"
		} else {
			getInt, _ := strconv.Atoi(get)
			formGet = formatNumberWithCommas(int64(getInt))
		}
		if remaining == "" {
			remaining = "----"
		} else {
			remaintInt, _ := strconv.Atoi(remaining)
			formRemain = formatNumberWithCommas(int64(remaintInt))
		}

		// Add row to contents
		contents = append(contents, Box{
			Type:   "box",
			Layout: "horizontal",
			// Height: "25px",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   name,
					Weight: "bold",
					Size:   "xs",
					Align:  "start",
				},
				Box{
					Type:   "text",
					Text:   formGet,
					Weight: "bold",
					Size:   "xs",
					Align:  "end",
					Color:  "#FF0000",
				},
				Box{
					Type:   "text",
					Text:   formRemain,
					Weight: "bold",
					Size:   "xs",
					Align:  "end",
					Color:  "#0000FF",
				},
			},
		})
		contents = append(contents, Box{
			Type:   "box",
			Layout: "vertical",
			Contents: []interface{}{
				Box{
					Type: "separator",
				},
			},
		})

		// Increment row count
		rowCount++

		// Check if we reached 100 rows and need to create a new bubble
		if rowCount == maxRows {
			round, _, _, _, _, _, _, localCommand, _, _, _, _ := GetLocalVar()
			hero := Box{
				Type:            "box",
				Layout:          "vertical",
				Height:          "30px",
				Align:           "center",
				JustifyContent:  "center",
				BackgroundColor: "#FFD700",
				BorderColor:     "#000000", // สีทอง
				BorderWidth:     "2px",     // ความหนาเส้นขอบ
				CornerRadius:    "xl",
				// Spacing: "xl",
				Contents: []interface{}{
					Box{
						Type:           "text",
						Text:           fmt.Sprintf("ผลการแทง คู่%d (%s)", round, localCommand),
						Weight:         "bold",
						Size:           "md",
						Align:          "center",
						JustifyContent: "center",
					},
					Box{
						Type: "separator",
					},
				},
			}

			// Create body section with vertical layout
			body := Box{
				Type:     "box",
				Layout:   "vertical",
				Contents: contents, // Add dynamically created contents
			}

			// Footer section with simple text
			// uriAction := Action{
			// 	Type: "uri",
			// 	URI:  liffDev101, // LIFF URL
			// }
			footer := Box{
				Type:   "box",
				Layout: "horizontal",
				Contents: []interface{}{
					Box{
						Type:  "text",
						Text:  houseName,
						Size:  "xxs",
						Align: "center",  // Footer text centered
						Color: "#000000", // Dark gray color
						// Action:          &uriAction,
						BackgroundColor: "",
					},
				},
			}

			// Create bubble containing hero, body, and footer
			bubble := Bubble{
				Type:   "bubble",
				Size:   "giga",
				Hero:   hero,
				Body:   body,
				Footer: footer,
			}

			// Add the bubble to the carousel
			flexMessages = append(flexMessages, bubble)

			// Reset contents for the next bubble
			contents = []interface{}{
				Box{
					Type:   "box",
					Layout: "horizontal",
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   "ชื่อ",
							Weight: "bold",
							Size:   "xs",
							Align:  "start",
						},
						Box{
							Type:   "text",
							Text:   "แดง",
							Weight: "bold",
							Size:   "xs",
							Align:  "end",
							Color:  "#FF0000",
						},
						Box{
							Type:   "text",
							Text:   "น้ำเงิน",
							Weight: "bold",
							Size:   "xs",
							Align:  "end",
							Color:  "#0000FF",
						},
					},
				},
				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type: "separator",
						},
					},
				},
			}
			rowCount = 0 // Reset row count for the next bubble
		}
	}

	// After the loop, if there are any remaining rows that weren't added to a bubble, create a final bubble
	if rowCount >= 0 {
		round, _, _, _, _, _, _, local_command, _, _, _, _ := GetLocalVar()
		hero := Box{
			Type:   "box",
			Layout: "vertical",
			Height: "30px",
			// Spacing:         "sm",
			BackgroundColor: "#FFD700",
			BorderColor:     "#000000", // สีทอง
			BorderWidth:     "2px",     // ความหนาเส้นขอบ
			CornerRadius:    "sm",

			// BackgroundColor: "#FF0000",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   fmt.Sprintf("ผลการแทง คู่%d (%s)", round, local_command),
					Weight: "bold",
					Size:   "md",
					Align:  "center",
					Color:  "#000000",
					// Margin: "10px",
				},
				// Box{
				// 	Margin: "5px",
				// 	Type:   "separator",
				// },
			},
		}

		body := Box{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		}

		// uriAction := Action{
		// 	Type: "uri",
		// 	URI:  liffProfile, // LIFF URL
		// }
		footer := Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Box{
					Type:  "text",
					Text:  houseName,
					Size:  "xxs",
					Align: "center",  // Footer text centered
					Color: "#000000", // Dark gray color
					// Action:          &uriAction,
					BackgroundColor: "",
				},
			},
		}

		bubble := Bubble{
			Type:   "bubble",
			Size:   "giga",
			Hero:   hero,
			Body:   body,
			Footer: footer,
		}

		flexMessages = append(flexMessages, bubble)
	}

	// If no messages were created, provide a fallback message
	if len(flexMessages) == 0 {
		flexMessages = append(flexMessages, Bubble{
			Type: "bubble",
			Size: "giga",
			Body: Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type:   "text",
						Text:   "ไม่มีข้อมูลการเล่น",
						Weight: "bold",
						Size:   "md",
						Align:  "center",
					},
				},
			},
		})
	}

	// Create final flex message with carousel layout
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: flexMessages,
	}

	return &flexMessage, nil
}
func GenerateFlexMessage(te [][]string) ([]linebot.SendingMessage, error) {
	// Initialize variables
	var bubbles []*linebot.BubbleContainer
	var contents []linebot.FlexComponent
	rowCount := 0
	maxRows := 100 // Limit the rows to 100
	hasData := false

	// Header row (Name, Get, Remaining)
	contents = append(contents,
		&linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ชื่อ",
					Weight: "bold",
					Size:   "xs",
					Align:  "start",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "แดง",
					Weight: "bold",
					Size:   "xs",
					Align:  "end",
					Color:  "#FF0000",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "น้ำเงิน",
					Weight: "bold",
					Size:   "xs",
					Align:  "end",
					Color:  "#0000FF",
				},
			},
		},
		&linebot.SeparatorComponent{},
	)

	// Iterate through the 2D array to create dynamic rows
	for _, row := range te[1:] { // Skip the header row
		if rowCount >= maxRows {
			break
		}

		var formGet, formRemain string
		name := row[0]
		if name == "" {
			name = "----"
		}

		get, remaining, _ := extractNumbers(row[1])
		if get == "" {
			get = "----"
		} else {
			getInt, _ := strconv.Atoi(get)
			formGet = formatNumberWithCommas(int64(getInt))
			hasData = true
		}
		if remaining == "" {
			remaining = "----"
		} else {
			remaintInt, _ := strconv.Atoi(remaining)
			formRemain = formatNumberWithCommas(int64(remaintInt))
			hasData = true
		}

		// Add row to contents
		contents = append(contents,
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:   "text",
						Text:   name,
						Weight: "bold",
						Size:   "xs",
						Align:  "start",
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formGet,
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  "#FF0000",
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formRemain,
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  "#0000FF",
					},
				},
			},
			&linebot.SeparatorComponent{},
		)

		rowCount++

		// Create new bubble when reaching max rows
		if rowCount == maxRows {
			round, _, _, _, _, _, _, localCommand, _, _, _, _ := GetLocalVar()

			var heroText string
			if hasData {
				heroText = fmt.Sprintf("ผลการแทง คู่%d (%s)", round, localCommand)
			} else {
				heroText = "ปิดรับการเดิมพัน"
			}

			hero := &linebot.BoxComponent{
				Type:            "box",
				Layout:          "vertical",
				Height:          "30px",
				AlignItems:      "center",
				JustifyContent:  "center",
				BackgroundColor: "#FFD700",
				BorderColor:     "#000000",
				BorderWidth:     "2px",
				CornerRadius:    "xl",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:   "text",
						Text:   heroText,
						Weight: "bold",
						Size:   "md",
						Align:  "center",
					},
					&linebot.SeparatorComponent{},
				},
			}

			body := &linebot.BoxComponent{
				Type:     "box",
				Layout:   "vertical",
				Contents: contents,
			}

			footer := &linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:  "text",
						Text:  houseName,
						Size:  "xxs",
						Align: "center",
						Color: "#000000",
					},
				},
			}

			bubble := &linebot.BubbleContainer{
				Type:   "bubble",
				Size:   "giga",
				Hero:   hero,
				Body:   body,
				Footer: footer,
			}

			bubbles = append(bubbles, bubble)

			// Reset for next bubble
			contents = []linebot.FlexComponent{
				&linebot.BoxComponent{
					Type:   "box",
					Layout: "horizontal",
					Contents: []linebot.FlexComponent{
						&linebot.TextComponent{
							Type:   "text",
							Text:   "ชื่อ",
							Weight: "bold",
							Size:   "xs",
							Align:  "start",
						},
						&linebot.TextComponent{
							Type:   "text",
							Text:   "แดง",
							Weight: "bold",
							Size:   "xs",
							Align:  "end",
							Color:  "#FF0000",
						},
						&linebot.TextComponent{
							Type:   "text",
							Text:   "น้ำเงิน",
							Weight: "bold",
							Size:   "xs",
							Align:  "end",
							Color:  "#0000FF",
						},
					},
				},
				&linebot.SeparatorComponent{},
			}
			rowCount = 0
			hasData = false
		}
	}

	// Add remaining rows to final bubble
	if rowCount > 0 {
		round, _, _, _, _, _, _, local_command, _, _, _, _ := GetLocalVar()

		var heroText string
		if hasData {
			heroText = fmt.Sprintf("ผลการแทง คู่%d (%s)", round, local_command)
		} else {
			heroText = "ปิดรับการเดิมพัน"
		}

		hero := &linebot.BoxComponent{
			Type:            "box",
			Layout:          "vertical",
			Height:          "30px",
			BackgroundColor: "#FFD700",
			BorderColor:     "#000000",
			BorderWidth:     "2px",
			CornerRadius:    "sm",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:   "text",
					Text:   heroText,
					Weight: "bold",
					Size:   "md",
					Align:  "center",
					Color:  "#000000",
				},
			},
		}

		body := &linebot.BoxComponent{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		}

		footer := &linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:  "text",
					Text:  houseName,
					Size:  "xxs",
					Align: "center",
					Color: "#000000",
				},
			},
		}

		bubble := &linebot.BubbleContainer{
			Type:   "bubble",
			Size:   "giga",
			Hero:   hero,
			Body:   body,
			Footer: footer,
		}

		bubbles = append(bubbles, bubble)
	}

	// Fallback message if no data
	if len(bubbles) == 0 {
		bubbles = append(bubbles, &linebot.BubbleContainer{
			Type: "bubble",
			Size: "giga",
			Hero: &linebot.BoxComponent{
				Type:            "box",
				Layout:          "vertical",
				Height:          "30px",
				BackgroundColor: "#FFD700",
				BorderColor:     "#000000",
				BorderWidth:     "2px",
				CornerRadius:    "sm",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:   "text",
						Text:   "ปิดรับการเดิมพัน",
						Weight: "bold",
						Size:   "md",
						Align:  "center",
						Color:  "#000000",
					},
				},
			},
			Body: &linebot.BoxComponent{
				Type:   "box",
				Layout: "vertical",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type:   "text",
						Text:   "ไม่มีข้อมูล",
						Weight: "bold",
						Size:   "md",
						Align:  "center",
					},
				},
			},
		})
	}

	// Create final flex message
	flexMessage := linebot.NewFlexMessage(
		"ผลการแทง",
		&linebot.CarouselContainer{
			Type:     "carousel",
			Contents: bubbles,
		},
	)

	var messages []linebot.SendingMessage
	messages = append(messages, flexMessage)

	return messages, nil
}
func GenerateFlexMessageORIGINAL(te [][]string) (*FlexMessage, error) {
	// Initialize variables
	var flexMessages []interface{}
	var contents []interface{}
	rowCount := 0
	maxRows := 100 // Limit the rows to 100

	// Header row (Name, Get, Remaining)
	contents = append(contents, Box{
		Type:   "box",        // This is the container for the horizontal layout
		Layout: "horizontal", // Set the layout to horizontal
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "ชื่อ",
				Weight: "bold",
				Size:   "sm",
				Align:  "start",
			},
			Box{
				Type:   "text",
				Text:   "แดง",
				Weight: "bold",
				Size:   "sm",
				Align:  "end",
				Color:  "#FF0000",
			},
			Box{
				Type:   "text",
				Text:   "น้ำเงิน",
				Weight: "bold",
				Size:   "sm",
				Align:  "end",
				Color:  "#0000FF",
			},
		},
	})
	contents = append(contents, Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			Box{
				Type: "separator",
			},
		},
	})

	// Iterate through the 2D array to create dynamic rows based on input
	for _, row := range te[1:] { // Skip the header row
		if rowCount >= maxRows { // Stop processing if we reach 100 rows
			break
		}
		var formGet, formRemain string
		// If data is empty, replace it with "----"
		name := row[0]
		if name == "" {
			name = "----"
		}

		get, remaining, _ := extractNumbers(row[1])
		if get == "" {
			get = "----"
		} else {
			getInt, _ := strconv.Atoi(get)
			formGet = formatNumberWithCommas(int64(getInt))
		}
		if remaining == "" {
			remaining = "----"
		} else {
			remaintInt, _ := strconv.Atoi(remaining)
			formRemain = formatNumberWithCommas(int64(remaintInt))
		}

		// Add row to contents
		contents = append(contents, Box{
			Type:   "box",
			Layout: "horizontal",
			Height: "25px",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   name,
					Weight: "bold",
					Size:   "sm",
					Align:  "start",
				},
				Box{
					Type:   "text",
					Text:   formGet,
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
					Color:  "#FF0000",
				},
				Box{
					Type:   "text",
					Text:   formRemain,
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
					Color:  "#0000FF",
				},
			},
		})
		contents = append(contents, Box{
			Type:   "box",
			Layout: "vertical",
			Contents: []interface{}{
				Box{
					Type: "separator",
				},
			},
		})

		// Increment row count
		rowCount++

		// Check if we reached 100 rows and need to create a new bubble
		if rowCount == maxRows {
			round, sub, _, _, _, _, _, _, _, _, _, _ := GetLocalVar()
			hero := Box{
				Type:    "box",
				Layout:  "vertical",
				Height:  "50px",
				Spacing: "xl",
				Contents: []interface{}{
					Box{
						Type:   "text",
						Text:   fmt.Sprintf("สรุปการแทง# %d (%d)", round, sub),
						Weight: "bold",
						Size:   "xl",
						Align:  "center",
					},
					Box{
						Type: "separator",
					},
				},
			}

			// Create body section with vertical layout
			body := Box{
				Type:     "box",
				Layout:   "vertical",
				Contents: contents, // Add dynamically created contents
			}

			// Footer section with simple text
			// uriAction := Action{
			// 	Type: "uri",
			// 	URI:  liffDev101, // LIFF URL
			// }
			footer := Box{
				Type:   "box",
				Layout: "horizontal",
				Contents: []interface{}{
					Box{
						Type:  "text",
						Text:  houseName,
						Size:  "xs",
						Align: "center",  // Footer text centered
						Color: "#000000", // Dark gray color
						// Action:          &uriAction,
						BackgroundColor: "",
					},
				},
			}

			// Create bubble containing hero, body, and footer
			bubble := Bubble{
				Type:   "bubble",
				Size:   "giga",
				Hero:   hero,
				Body:   body,
				Footer: footer,
			}

			// Add the bubble to the carousel
			flexMessages = append(flexMessages, bubble)

			// Reset contents for the next bubble
			contents = []interface{}{
				Box{
					Type:   "box",
					Layout: "horizontal",
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   "ชื่อ",
							Weight: "bold",
							Size:   "sm",
							Align:  "start",
						},
						Box{
							Type:   "text",
							Text:   "แดง",
							Weight: "bold",
							Size:   "sm",
							Align:  "end",
							Color:  "#FF0000",
						},
						Box{
							Type:   "text",
							Text:   "น้ำเงิน",
							Weight: "bold",
							Size:   "sm",
							Align:  "end",
							Color:  "#0000FF",
						},
					},
				},
				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type: "separator",
						},
					},
				},
			}
			rowCount = 0 // Reset row count for the next bubble
		}
	}

	// After the loop, if there are any remaining rows that weren't added to a bubble, create a final bubble
	if rowCount >= 0 {
		round, sub, _, _, _, _, _, _, _, _, _, _ := GetLocalVar()
		hero := Box{
			Type:            "box",
			Layout:          "vertical",
			Height:          "52px",
			Spacing:         "sm",
			BackgroundColor: "#eeeeee",

			// BackgroundColor: "#FF0000",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   fmt.Sprintf("สรุปการแทง คู่ที่ %d (%d)", round, sub),
					Weight: "bold",
					Size:   "xl",
					Align:  "center",
					Color:  "#000000",
					Margin: "10px",
				},
				// Box{
				// 	Margin: "5px",
				// 	Type:   "separator",
				// },
			},
		}

		body := Box{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		}

		// uriAction := Action{
		// 	Type: "uri",
		// 	URI:  liffProfile, // LIFF URL
		// }
		footer := Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Box{
					Type:  "text",
					Text:  houseName,
					Size:  "xs",
					Align: "center",  // Footer text centered
					Color: "#000000", // Dark gray color
					// Action:          &uriAction,
					BackgroundColor: "",
				},
			},
		}

		bubble := Bubble{
			Type:   "bubble",
			Size:   "giga",
			Hero:   hero,
			Body:   body,
			Footer: footer,
		}

		flexMessages = append(flexMessages, bubble)
	}

	// If no messages were created, provide a fallback message
	if len(flexMessages) == 0 {
		flexMessages = append(flexMessages, Bubble{
			Type: "bubble",
			Size: "giga",
			Body: Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type:   "text",
						Text:   "ไม่มีข้อมูล",
						Weight: "bold",
						Size:   "xl",
						Align:  "center",
					},
				},
			},
		})
	}

	// Create final flex message with carousel layout
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: flexMessages,
	}

	return &flexMessage, nil
}

func GenerateFlexMessage2OLD(te [][]string, round int, mes string) (*FlexMessage, error) {
	// Split the message by newline characters

	// If the message contains more than 70 lines, take only the first 70
	limitRows := te[0:] // Skip the header row
	if len(limitRows) > 70 {
		limitRows = limitRows[:70] // Limit to first 70 rows
	}

	// Continue with the rest of the code to generate the Flex message...

	// Header row (Name, Get, Remaining)
	contents := []interface{}{
		Box{
			Type:   "box",        // This is the container for the horizontal layout
			Layout: "horizontal", // Set the layout to horizontal
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   "ชื่อ",
					Weight: "bold",
					Size:   "sm",
					Align:  "start",
				},
				Box{
					Type:   "text",
					Text:   "ทุน",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				Box{
					Type:   "text",
					Text:   "ได้/เสีย",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				Box{
					Type:   "text",
					Text:   "คงเหลือ",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
			},
		},
		Box{
			Type: "separator", // Separator between the elements
		},
	}

	// Iterate through the first 70 rows (or fewer if less than 70 rows)
	for i := 1; i <= 1; i++ {
		for _, row := range limitRows[1:] { // Skipping the first header row
			// Extract Name, Get, and Remaining
			name := row[0]                                    // Name
			bal, get, remaining, _ := extractNumbers2(row[1]) // Get

			// Convert values and handle potential errors
			getInt, err := strconv.Atoi(get)
			if err != nil {
				return nil, fmt.Errorf("invalid 'get' value: %v", err)
			}

			remainingInt, err := strconv.Atoi(remaining)
			if err != nil {
				fmt.Println("Error converting remaining:", err)
			}

			balInt, err := strconv.Atoi(bal)
			if err != nil {
				fmt.Println("Error converting remaining:", err)
			}

			// Get color for `get` and `remaining` based on their values
			BalColor := getColor2(balInt)
			getColorValue := getColor(getInt)
			remainingColorValue := getColor2(remainingInt)

			// Add row contents to the Flex message
			contents = append(contents, Box{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []interface{}{
					Box{
						Type: "text",
						Text: name,
						// Weight: "bold",
						Size:  "sm",
						Align: "start",
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(balInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  BalColor,
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(getInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  getColorValue,
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(remainingInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  remainingColorValue,
					},
				},
			})
			contents = append(contents, Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type: "separator",
					},
				},
			})
		}
	}

	// Create hero section
	var cc = "#0000ff"
	var bc = "#ffffff"
	if mes == "เสมอ" {
		cc = "#00db00"
		bc = "#ffffff"
	} else if mes == "แดงชนะ" {
		cc = "#ff0000"
	}

	hero := Box{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: cc,
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "คู่ที่  " + strconv.Itoa(round) + " " + mes,
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",
				Color:  bc,
			},
		},
	}

	// Create body section with vertical layout
	body := Box{
		Type:     "box",
		Layout:   "vertical",
		Contents: contents,
	}

	// Footer section
	// uriAction := Action{
	// 	Type: "uri",
	// 	URI:  liffProfile,
	// }
	footer := Box{
		Type:   "box",
		Layout: "horizontal",
		Contents: []interface{}{
			Box{
				Type:  "text",
				Text:  houseName,
				Size:  "xs",
				Align: "center",
				Color: "#000000",
				// Action:          &uriAction,
				BackgroundColor: "",
			},
		},
	}

	// Create Flex message
	bubble := Bubble{
		Type:   "bubble",
		Size:   "giga",
		Hero:   hero,
		Body:   body,
		Footer: footer,
	}

	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: []interface{}{bubble},
	}

	return &flexMessage, nil
}
func GenerateFlexMessage2(te [][]string, round int, mes string) ([]linebot.SendingMessage, error) {
	// Limit rows to 70
	limitRows := te[0:]
	if len(limitRows) > 110 {
		limitRows = limitRows[:110]
	}

	// Initialize contents with header row
	contents := []linebot.FlexComponent{
		&linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ชื่อ",
					Weight: "bold",
					Size:   "sm",
					// Align:  "start",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ต้น",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ผล",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "คงเหลือ",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
			},
		},
		&linebot.SeparatorComponent{},
	}

	// Process each row
	for _, row := range limitRows[1:] {
		name := row[0]
		bal, get, remaining, _ := extractNumbers2(row[1])

		// Convert values
		getInt, err := strconv.Atoi(get)
		if err != nil {
			return nil, fmt.Errorf("invalid 'get' value: %v", err)
		}

		remainingInt, _ := strconv.Atoi(remaining)
		balInt, _ := strconv.Atoi(bal)

		// Get colors
		balColor := getColor2(balInt)
		getColorValue := getColor(getInt)
		remainingColorValue := getColor2(remainingInt)

		// Add row to contents
		contents = append(contents,
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type: "text",
						Text: name,
						Size: "xs",
						// Align: "start",
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(balInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  balColor,
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(getInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  getColorValue,
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(remainingInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  remainingColorValue,
					},
				},
			},
			&linebot.SeparatorComponent{},
		)
	}

	// Set hero colors based on result
	var cc = "#0000ff" // Default blue
	var bc = "#ffffff" // Default white
	if mes == "เสมอ" {
		cc = "#FFD700"
	} else if mes == "แดงชนะ" {
		cc = "#ff0000" // Red for red wins
	}

	// Create hero section
	hero := &linebot.BoxComponent{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: cc,

		BorderColor:  "#000000",
		BorderWidth:  "2px",
		CornerRadius: "xl",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:   "text",
				Text:   "คู่  " + strconv.Itoa(round) + " " + mes,
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",

				Color: bc,
			},
		},
	}

	// Create body section
	body := &linebot.BoxComponent{
		Type:     "box",
		Layout:   "vertical",
		Contents: contents,
	}

	// Create footer section
	footer := &linebot.BoxComponent{
		Type:   "box",
		Layout: "horizontal",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:  "text",
				Text:  houseName,
				Size:  "xs",
				Align: "center",
				Color: "#000000",
			},
		},
	}

	// Create bubble container
	bubble := &linebot.BubbleContainer{
		Type:   "bubble",
		Size:   "giga",
		Hero:   hero,
		Body:   body,
		Footer: footer,
	}

	// Create flex message
	flexMessage := linebot.NewFlexMessage(
		"ผลการแทง",
		&linebot.CarouselContainer{
			Type:     "carousel",
			Contents: []*linebot.BubbleContainer{bubble},
		},
	)

	var messages []linebot.SendingMessage
	messages = append(messages, flexMessage)

	return messages, nil
}
func GenerateFlexMessage2p(te [][]string, round int, mes string) ([]linebot.SendingMessage, error) {
	// Limit rows only 110+
	limitRows := te[0:]
	if len(limitRows) > 110 {
		limitRows = limitRows[109:]
	} else {
		fmt.Println("NOT ENOUGH 110")
		limitRows = te[0:1]
	}

	// Initialize contents with header row
	contents := []linebot.FlexComponent{
		&linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ชื่อ",
					Weight: "bold",
					Size:   "sm",
					// Align:  "start",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ต้น",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "ผล",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
				&linebot.TextComponent{
					Type:   "text",
					Text:   "คงเหลือ",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
			},
		},
		&linebot.SeparatorComponent{},
	}

	// Process each row
	for _, row := range limitRows[1:] {
		name := row[0]
		bal, get, remaining, _ := extractNumbers2(row[1])

		// Convert values
		getInt, err := strconv.Atoi(get)
		if err != nil {
			return nil, fmt.Errorf("invalid 'get' value: %v", err)
		}

		remainingInt, _ := strconv.Atoi(remaining)
		balInt, _ := strconv.Atoi(bal)

		// Get colors
		balColor := getColor2(balInt)
		getColorValue := getColor(getInt)
		remainingColorValue := getColor2(remainingInt)

		// Add row to contents
		contents = append(contents,
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type: "text",
						Text: name,
						Size: "xs",
						// Align: "start",
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(balInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  balColor,
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(getInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  getColorValue,
					},
					&linebot.TextComponent{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(remainingInt)),
						Weight: "bold",
						Size:   "xs",
						Align:  "end",
						Color:  remainingColorValue,
					},
				},
			},
			&linebot.SeparatorComponent{},
		)
	}

	// Set hero colors based on result
	var cc = "#0000ff" // Default blue
	var bc = "#ffffff" // Default white
	if mes == "เสมอ" {
		cc = "#FFD700"
	} else if mes == "แดงชนะ" {
		cc = "#ff0000" // Red for red wins
	}

	// Create hero section
	hero := &linebot.BoxComponent{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: cc,

		BorderColor:  "#000000",
		BorderWidth:  "2px",
		CornerRadius: "xl",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:   "text",
				Text:   "คู่  " + strconv.Itoa(round-1) + " " + mes + " (ต่อ)",
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",

				Color: bc,
			},
		},
	}

	// Create body section
	body := &linebot.BoxComponent{
		Type:     "box",
		Layout:   "vertical",
		Contents: contents,
	}

	// Create footer section
	footer := &linebot.BoxComponent{
		Type:   "box",
		Layout: "horizontal",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:  "text",
				Text:  houseName,
				Size:  "xs",
				Align: "center",
				Color: "#000000",
			},
		},
	}

	// Create bubble container
	bubble := &linebot.BubbleContainer{
		Type:   "bubble",
		Size:   "giga",
		Hero:   hero,
		Body:   body,
		Footer: footer,
	}

	// Create flex message
	flexMessage := linebot.NewFlexMessage(
		"ผลการแทง",
		&linebot.CarouselContainer{
			Type:     "carousel",
			Contents: []*linebot.BubbleContainer{bubble},
		},
	)

	var messages []linebot.SendingMessage
	messages = append(messages, flexMessage)

	return messages, nil
}
func GenerateFlexMessage22OLD(te [][]string, round int, mes string) (*FlexMessage, error) {
	// Split the message by newline characters

	// If the message contains more than 70 lines, take only the first 70
	limitRows := te[0:] // Skip the header row
	if len(limitRows) > 70 {
		limitRows = limitRows[:70] // Limit to first 70 rows
	}

	// Continue with the rest of the code to generate the Flex message...

	// Header row (Name, Get, Remaining)
	contents := []interface{}{
		Box{
			Type:   "box",        // This is the container for the horizontal layout
			Layout: "horizontal", // Set the layout to horizontal
			Contents: []interface{}{
				Box{
					Type: "text",
					Text: "ชื่อ",
					// Weight: "bold",
					Size:  "sm",
					Align: "start",
				},
				Box{
					Type: "text",
					Text: "ทุน",
					// Weight: "bold",
					Size:  "sm",
					Align: "end",
				},
				Box{
					Type: "text",
					Text: "ได้/เสีย",
					// Weight: "bold",
					Size:  "sm",
					Align: "end",
				},
				Box{
					Type: "text",
					Text: "คงเหลือ",
					// Weight: "bold",
					Size:  "sm",
					Align: "end",
				},
			},
		},
		Box{
			Type: "separator", // Separator between the elements
		},
	}

	// Iterate through the first 70 rows (or fewer if less than 70 rows)
	for i := 1; i <= 1; i++ {
		for _, row := range limitRows[0:] { // Skipping the first header row
			// Extract Name, Get, and Remaining
			name := row[0]                                    // Name
			bal, get, remaining, _ := extractNumbers2(row[1]) // Get

			// Convert values and handle potential errors
			getInt, err := strconv.Atoi(get)
			if err != nil {
				return nil, fmt.Errorf("invalid 'get' value: %v", err)
			}

			remainingInt, err := strconv.Atoi(remaining)
			if err != nil {
				fmt.Println("Error converting remaining:", err)
			}

			balInt, err := strconv.Atoi(bal)
			if err != nil {
				fmt.Println("Error converting remaining:", err)
			}

			// Get color for `get` and `remaining` based on their values
			BalColor := getColor2(balInt)
			getColorValue := getColor(getInt)
			remainingColorValue := getColor2(remainingInt)

			// Add row contents to the Flex message
			contents = append(contents, Box{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []interface{}{
					Box{
						Type: "text",
						Text: name,
						// Weight: "bold",
						Size:  "sm",
						Align: "start",
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(balInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  BalColor,
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(getInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  getColorValue,
					},
					Box{
						Type:   "text",
						Text:   formatNumberWithCommas(int64(remainingInt)),
						Weight: "bold",
						Size:   "sm",
						Align:  "end",
						Color:  remainingColorValue,
					},
				},
			})
			contents = append(contents, Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type: "separator",
					},
				},
			})
		}
	}

	// Create hero section
	// var cc = "#00db00"
	// var bc = "#ffffff"
	if mes == "เสมอ" {
		// cc = "#00db00"
		// bc = "#ffffff"
	} else if mes == "แดงชนะ" {
		// cc = "#ff0000"
	}

	hero := Box{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: "#FFD700",
		BorderColor:     "#000000", // สีทอง
		BorderWidth:     "2px",     // ความหนาเส้นขอบ
		CornerRadius:    "sm",
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "ยอดได้เสีย",
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",
				Color:  "#000000",
			},
		},
	}

	// Create body section with vertical layout
	body := Box{
		Type:     "box",
		Layout:   "vertical",
		Contents: contents,
	}

	// Footer section
	// uriAction := Action{
	// 	Type: "uri",
	// 	URI:  liffProfile,
	// }
	footer := Box{
		Type:   "box",
		Layout: "horizontal",
		Contents: []interface{}{
			Box{
				Type:  "text",
				Text:  houseName,
				Size:  "xs",
				Align: "center",
				Color: "#000000",
				// Action:          &uriAction,
				BackgroundColor: "",
			},
		},
	}

	// Create Flex message
	bubble := Bubble{
		Type:   "bubble",
		Size:   "giga",
		Hero:   hero,
		Body:   body,
		Footer: footer,
	}

	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: []interface{}{bubble},
	}

	return &flexMessage, nil
}
func GenerateFlexMessage22(te [][]string, round int, mes string) ([]linebot.SendingMessage, error) {
	// Limit rows to 70
	limitRows := te[0:]
	if len(limitRows) > 130 {
		limitRows = limitRows[:130]
	}

	// Create hero section
	hero := &linebot.BoxComponent{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: "#FFD700",
		BorderColor:     "#000000",
		BorderWidth:     "2px",
		CornerRadius:    "sm",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:   "text",
				Text:   "ยอดได้เสีย",
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",
				Color:  "#000000",
			},
		},
	}

	// Create footer section
	footer := &linebot.BoxComponent{
		Type:   "box",
		Layout: "horizontal",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type: "text",
				Text: houseName,
				// Size:  "xs",
				Align: "center",
				// Color: "#000000",
			},
		},
	}

	// Initialize contents with header row
	contents := []linebot.FlexComponent{
		&linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type: "text",
					Text: "ชื่อ",
					Size: "sm",
					// Align: "start",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "ทุน",
					Size:  "sm",
					Align: "end",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "ได้/เสีย",
					Size:  "sm",
					Align: "end",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "คงเหลือ",
					Size:  "sm",
					Align: "end",
				},
			},
		},
		&linebot.SeparatorComponent{},
	}

	// Process each row
	for _, row := range limitRows {
		name := row[0]
		bal, get, remaining, _ := extractNumbers2(row[1])

		balInt, _ := strconv.Atoi(bal)
		getInt, _ := strconv.Atoi(get)
		remainingInt, _ := strconv.Atoi(remaining)

		balColor := getColor2(balInt)
		getColorValue := getColor(getInt)
		remainingColorValue := getColor2(remainingInt)

		contents = append(contents,
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type: "text",
						Text: name,
						Size: "xs",
						// Align: "start",
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(balInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: balColor,
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(getInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: getColorValue,
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(remainingInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: remainingColorValue,
					},
				},
			},
			&linebot.SeparatorComponent{},
		)
	}

	// Create the bubble container
	bubble := &linebot.BubbleContainer{
		Type: "bubble",
		Size: "giga",
		Hero: hero,
		Body: &linebot.BoxComponent{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		},
		Footer: footer,
	}

	// Create Flex message
	flexMessage := linebot.NewFlexMessage(
		"ผลลัพธ์ประจำวัน",
		&linebot.CarouselContainer{
			Type:     "carousel",
			Contents: []*linebot.BubbleContainer{bubble},
		},
	)

	var messages []linebot.SendingMessage
	messages = append(messages, flexMessage)

	return messages, nil
}
func GenerateFlexMessage22V2(te [][]string, round int, mes string) ([]linebot.SendingMessage, error) {
	// Limit rows to 70
	limitRows := te[0:]
	if len(te) > 130 {
		end := len(te)
		if end > 260 {
			end = 260
		}
		limitRows = te[130:end]
	} else {
		limitRows = [][]string{{"ข้อมูลไม่เพียงพอ", "0/0/0"}}

	}
	// Create hero section
	hero := &linebot.BoxComponent{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: "#FFD700",
		BorderColor:     "#000000",
		BorderWidth:     "2px",
		CornerRadius:    "sm",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type:   "text",
				Text:   "ยอดได้เสีย(130+)",
				Weight: "bold",
				Size:   "xxl",
				Align:  "center",
				Margin: "4px",
				Color:  "#000000",
			},
		},
	}

	// Create footer section
	footer := &linebot.BoxComponent{
		Type:   "box",
		Layout: "horizontal",
		Contents: []linebot.FlexComponent{
			&linebot.TextComponent{
				Type: "text",
				Text: houseName,
				// Size:  "xs",
				Align: "center",
				// Color: "#000000",
			},
		},
	}

	// Initialize contents with header row
	contents := []linebot.FlexComponent{
		&linebot.BoxComponent{
			Type:   "box",
			Layout: "horizontal",
			Contents: []linebot.FlexComponent{
				&linebot.TextComponent{
					Type: "text",
					Text: "ชื่อ",
					Size: "sm",
					// Align: "start",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "ทุน",
					Size:  "sm",
					Align: "end",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "ได้/เสีย",
					Size:  "sm",
					Align: "end",
				},
				&linebot.TextComponent{
					Type:  "text",
					Text:  "คงเหลือ",
					Size:  "sm",
					Align: "end",
				},
			},
		},
		&linebot.SeparatorComponent{},
	}

	// Process each row
	for _, row := range limitRows {
		name := row[0]
		bal, get, remaining, _ := extractNumbers2(row[1])

		balInt, _ := strconv.Atoi(bal)
		getInt, _ := strconv.Atoi(get)
		remainingInt, _ := strconv.Atoi(remaining)

		balColor := getColor2(balInt)
		getColorValue := getColor(getInt)
		remainingColorValue := getColor2(remainingInt)

		contents = append(contents,
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Height: "22px",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{
						Type: "text",
						Text: name,
						Size: "xs",
						// Align: "start",
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(balInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: balColor,
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(getInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: getColorValue,
					},
					&linebot.TextComponent{
						Type: "text",
						Text: formatNumberWithCommas(int64(remainingInt)),
						// Weight: "bold",
						Size:  "xs",
						Align: "end",
						Color: remainingColorValue,
					},
				},
			},
			&linebot.SeparatorComponent{},
		)
	}

	// Create the bubble container
	bubble := &linebot.BubbleContainer{
		Type: "bubble",
		Size: "giga",
		Hero: hero,
		Body: &linebot.BoxComponent{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		},
		Footer: footer,
	}

	// Create Flex message
	flexMessage := linebot.NewFlexMessage(
		"ผลลัพธ์ประจำวัน",
		&linebot.CarouselContainer{
			Type:     "carousel",
			Contents: []*linebot.BubbleContainer{bubble},
		},
	)

	var messages []linebot.SendingMessage
	messages = append(messages, flexMessage)

	return messages, nil
}
func GenerateFlexMessageROLD(data string) (*FlexMessage, error) {
	// Split the input string into lines based on newline character

	lines := strings.Split(data, "\n")
	if data == "----" {
		lines = []string{"-//,//0"} // Correct slice initialization syntax
	}

	// Hero section (shared for each bubble)
	hero := Box{
		Type:            "box",
		Layout:          "vertical", // Vertical layout
		Height:          "50px",     // Set the container height to limit the image size
		Spacing:         "md",       // Medium spacing between the contents
		BackgroundColor: "#FFD700",
		BorderColor:     "#000000", // สีทอง
		BorderWidth:     "2px",     // ความหนาเส้นขอบ
		CornerRadius:    "sm",
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "ยอดคงเหลือ", // The text to display
				Weight: "bold",       // Make the text bold
				Size:   "xxl",        // Set text size to 'xl'
				Align:  "center",     // Center-align the text
				Color:  "#000000",
				Margin: "4px",
			},
		},
	}

	// Footer section with simple text (shared for each bubble)
	// uriAction := Action{
	// 	Type: "uri",
	// 	URI:  liffProfile, // LIFF URL
	// }
	footer := Box{
		Type:   "box",
		Layout: "horizontal",
		Contents: []interface{}{
			Box{
				Type:  "text",
				Text:  houseName,
				Size:  "xs",
				Align: "center",  // Footer text centered
				Color: "#000000", // Dark gray color
				// Action:          &uriAction,

				BackgroundColor: "",
			},
		},
	}

	// Create an array to hold multiple bubbles
	var bubbles []interface{}

	// Initialize the contents and row counter for the first bubble
	contents := []interface{}{
		Box{
			Type:   "box",        // This is the container for the horizontal layout
			Layout: "horizontal", // Set the layout to horizontal
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   "ชื่อ",
					Weight: "bold",
					Size:   "sm",
					Align:  "start",
				},

				Box{
					Type:   "text",
					Text:   "คงเหลือ",
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
				},
			},
		},
		Box{
			Type:  "separator",
			Color: "#FFD700", // สีทอง (Gold)
		},
	}

	// Row counters
	rowCount := 0
	totalRowCount := 0

	// Iterate through each line to create dynamic rows based on input
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split the line into name and balance using the //,// delimiter
		parts := strings.Split(line, "//,//")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in line: %v", line)
		}

		name := parts[0]    // Extract Name
		balance := parts[1] // Extract Balance

		// Extract balance as integer and format with commas
		balanceInt, err := strconv.Atoi(balance)
		if err != nil {
			return nil, fmt.Errorf("invalid balance value: %v", err)
		}
		balanceFormatted := formatNumberWithCommas(int64(balanceInt))

		// Get color for balance based on its value
		balanceColor := getColor2(balanceInt) // Get color for 'balance'

		// Add player information to contents
		contents = append(contents, Box{
			Type:   "box",
			Layout: "horizontal",
			Height: "22px",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   name,
					Weight: "bold",
					Size:   "sm",
					Align:  "start",
				},
				Box{
					Type:   "text",
					Text:   balanceFormatted,
					Weight: "bold",
					Size:   "sm",
					Align:  "end",
					Color:  balanceColor,
				},
			},
		})
		contents = append(contents, Box{
			Type:   "box",
			Layout: "vertical", // Corrected layout to "vertical"
			Contents: []interface{}{
				Box{
					Type:  "separator",
					Color: "#FFD700", // สีทอง (Gold)
				},
			},
		})

		// Increment row count
		rowCount++
		totalRowCount++

		// If the row count exceeds 40, create a new bubble
		if rowCount >= 120 || totalRowCount >= 120 {
			// Create new bubble with the current contents
			body := Box{
				Type:     "box",
				Layout:   "vertical",
				Contents: contents, // Add dynamically created contents
			}

			bubble := Bubble{
				Type:   "bubble",
				Size:   "giga", // Smaller bubble size
				Hero:   hero,   // Hero section with image
				Body:   body,   // Body with the dynamic content
				Footer: footer, // Footer with the powered-by text
			}

			// Add the new bubble to the bubbles array
			bubbles = append(bubbles, bubble)

			// Reset contents and row count for the next bubble
			contents = []interface{}{
				Box{
					Type:   "box",        // This is the container for the horizontal layout
					Layout: "horizontal", // Set the layout to horizontal
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   "ชื่อ",
							Weight: "bold",
							Size:   "sm",
							Align:  "start",
						},

						Box{
							Type:   "text",
							Text:   "คงเหลือ",
							Weight: "bold",
							Size:   "sm",
							Align:  "end",
						},
					},
				},
				Box{
					Type: "separator", // Separator between the elements
				},
			}
			rowCount = 0 // Reset the row counter for the next bubble

			// Stop if 120 rows have been processed
			if totalRowCount >= 120 {
				break
			}
		}
	}

	// If there are any remaining rows, add them to the last bubble
	if rowCount > 0 && totalRowCount < 120 {
		body := Box{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents, // Add remaining contents
		}
		bubble := Bubble{
			Type:   "bubble",
			Size:   "giga",
			Hero:   hero,
			Body:   body,
			Footer: footer,
		}

		bubbles = append(bubbles, bubble)
	}

	// Create Flex message with carousel containing all the bubbles
	flexMessage := FlexMessage{
		Type:     "carousel", // Carousel layout, which can hold multiple bubbles
		Contents: bubbles,    // Add all bubbles to carousel
	}

	return &flexMessage, nil
}
func GenerateFlexMessageR(data string) ([]linebot.SendingMessage, error) {
	lines := strings.Split(data, "\n")
	if data == "----" {
		lines = []string{"-//,//0"}
	}

	type Entry struct {
		Name   string
		Amount int
	}

	var positives, negatives []Entry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "//,//")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in line: %v", line)
		}

		name := parts[0]
		amount, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid balance: %v", parts[1])
		}

		if amount >= 0 {
			positives = append(positives, Entry{name, amount})
		} else {
			negatives = append(negatives, Entry{name, amount})
		}
	}

	buildBubbles := func(entries []Entry, title string) *linebot.FlexMessage {
		var bubbles []*linebot.BubbleContainer
		rowCount := 0
		totalRowCount := 0
		firstBubble := true

		contents := []linebot.FlexComponent{
			&linebot.BoxComponent{
				Type:   "box",
				Layout: "horizontal",
				Contents: []linebot.FlexComponent{
					&linebot.TextComponent{Type: "text", Text: "ชื่อ", Size: "sm"},
					&linebot.TextComponent{Type: "text", Text: "คงเหลือ", Size: "sm", Align: "end"},
				},
			},
			&linebot.SeparatorComponent{Color: "#FFD700"},
		}

		for _, entry := range entries {
			formatted := formatNumberWithCommas(int64(entry.Amount))
			color := getColor2(entry.Amount)

			contents = append(contents,
				&linebot.BoxComponent{
					Type:   "box",
					Layout: "horizontal",
					Height: "22px",
					Contents: []linebot.FlexComponent{
						&linebot.TextComponent{Type: "text", Text: entry.Name, Size: "sm"},
						&linebot.TextComponent{Type: "text", Text: formatted, Size: "sm", Align: "end", Color: color},
					},
				},
				&linebot.SeparatorComponent{},
			)

			rowCount++
			totalRowCount++

			if (rowCount >= 50 && firstBubble) || rowCount >= 54 || totalRowCount >= 212 {
				bubble := &linebot.BubbleContainer{
					Type: "bubble",
					Size: "giga",
					Body: &linebot.BoxComponent{
						Type:     "box",
						Layout:   "vertical",
						Contents: contents,
					},
				}

				if firstBubble {
					bubble.Hero = &linebot.BoxComponent{
						Type:            "box",
						Layout:          "vertical",
						Height:          "50px",
						BackgroundColor: "#FFD700",
						BorderColor:     "#000000",
						BorderWidth:     "2px",
						CornerRadius:    "sm",
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{
								Type:   "text",
								Text:   title,
								Weight: "bold",
								Size:   "xxl",
								Align:  "center",
								Color:  "#000000",
							},
						},
					}
					bubble.Footer = &linebot.BoxComponent{
						Type:   "box",
						Layout: "horizontal",
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{Type: "text", Text: houseName, Align: "center", Color: "#000000"},
						},
					}
					firstBubble = false
				}

				bubbles = append(bubbles, bubble)

				// Reset for next bubble
				contents = []linebot.FlexComponent{
					&linebot.BoxComponent{
						Type:   "box",
						Layout: "horizontal",
						Contents: []linebot.FlexComponent{
							&linebot.TextComponent{Type: "text", Text: "ชื่อ", Size: "sm"},
							&linebot.TextComponent{Type: "text", Text: "คงเหลือ", Size: "sm", Align: "end"},
						},
					},
					&linebot.SeparatorComponent{Color: "#FFD700"},
				}
				rowCount = 0
			}
		}

		if rowCount > 0 {
			bubble := &linebot.BubbleContainer{
				Type: "bubble",
				Size: "giga",
				Body: &linebot.BoxComponent{
					Type:     "box",
					Layout:   "vertical",
					Contents: contents,
				},
			}
			if firstBubble {
				bubble.Hero = &linebot.BoxComponent{
					Type:            "box",
					Layout:          "vertical",
					Height:          "50px",
					BackgroundColor: "#FFD700",
					BorderColor:     "#000000",
					BorderWidth:     "2px",
					CornerRadius:    "sm",
					Contents: []linebot.FlexComponent{
						&linebot.TextComponent{
							Type:   "text",
							Text:   title,
							Weight: "bold",
							Size:   "xxl",
							Align:  "center",
							Color:  "#000000",
						},
					},
				}
				bubble.Footer = &linebot.BoxComponent{
					Type:   "box",
					Layout: "horizontal",
					Contents: []linebot.FlexComponent{
						&linebot.TextComponent{Type: "text", Text: houseName, Align: "center", Color: "#000000"},
					},
				}
			}
			bubbles = append(bubbles, bubble)
		}

		return linebot.NewFlexMessage(title, &linebot.CarouselContainer{Type: "carousel", Contents: bubbles})
	}

	var messages []linebot.SendingMessage
	if len(positives) > 0 {
		messages = append(messages, buildBubbles(positives, "✅ ยอดบวก"))
	}
	if len(negatives) > 0 {
		messages = append(messages, buildBubbles(negatives, "❌ ยอดลบ"))
	}

	return messages, nil
}

type Hero struct {
	Type        string  `json:"type"`
	URL         string  `json:"url,omitempty"`
	Size        string  `json:"size,omitempty"`
	AspectRatio string  `json:"aspectRatio,omitempty"`
	AspectMode  string  `json:"aspectMode,omitempty"`
	Action      *Action `json:"action,omitempty"`
	Contents    *Box    `json:"contents,omitempty"`
}

func GenerateFlexMessageRE() (*FlexMessage, error) {
	// Hero section (vertical box with text)
	hero := Box{
		Type:            "box",
		Layout:          "vertical",
		Height:          "50px",
		Spacing:         "md",
		BackgroundColor: "#FFD700", // Set background color to green
		BorderColor:     "#000000", // สีทอง
		BorderWidth:     "2px",     // ความหนาเส้นขอบ
		CornerRadius:    "sm",
		Contents: []interface{}{
			Box{
				Type:   "text",
				Text:   "สรุปผล",  // The text to display
				Weight: "bold",    // Make the text bold
				Size:   "xxl",     // Set text size to 'xxl'
				Align:  "center",  // Center-align the text
				Color:  "#000000", // Text color set to black
				Margin: "4px",
			},
		},
	}

	// Hero image setup
	heroImage := Box{
		Type: "image",
		URL:  "https://i.postimg.cc/zBF9pBkh/RE.jpg",
		Size: "full",

		Action: &Action{
			Type: "uri",
			URI:  "https://line.me/",
		},
	}

	// Combine the text and image box to form the hero
	heroSection := Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			hero,      // Text part
			heroImage, // Image part
		},
	}

	// Bubble structure
	bubble := Bubble{
		Type: "bubble",
		Hero: heroSection, // Use the combined hero section
	}

	// Create FlexMessage structure
	flexMessage := FlexMessage{
		Type:     "carousel",            // Carousel format to allow multiple bubbles
		Contents: []interface{}{bubble}, // Wrap the bubble in a slice to make it []interface{}
	}

	return &flexMessage, nil
}

func GenerateFlexHomeOLD() *FlexMessage {
	flex := &FlexMessage{
		Type: "carousel",
		Contents: []interface{}{
			// First Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "RealTime",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "70%",
									"backgroundColor": "#0D8186",
									"height":          "6px",
								},
							},
							"backgroundColor": "#9FD8E36E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#0000FF",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "action",
					"uri":   liffRealTime,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// Second Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "เคลียร์ยอด",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "30%",
									"backgroundColor": "#DE5658",
									"height":          "6px",
								},
							},
							"backgroundColor": "#FAD2A76E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#FF0000",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "CLEAR",
					"uri":   liffClear,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// Third Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "ผลรวมทุกคู่",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "70%",
									"backgroundColor": "#0D8186",
									"height":          "6px",
								},
							},
							"backgroundColor": "#9FD8E36E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#0000FF",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "CREDIT",
					"uri":   liffSummary,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "หลังบ้าน",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "30%",
									"backgroundColor": "#DE5658",
									"height":          "6px",
								},
							},
							"backgroundColor": "#FAD2A76E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#FF0000",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "HOME",
					"uri":   liffHQ,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// map[string]interface{}{
			// 	"type": "bubble",
			// 	"size": "nano",
			// 	"header": map[string]interface{}{
			// 		"type":   "box",
			// 		"layout": "vertical",
			// 		"contents": []interface{}{
			// 			map[string]interface{}{
			// 				"type":    "text",
			// 				"text":    "Game",
			// 				"color":   "#ffffff",
			// 				"align":   "start",
			// 				"size":    "md",
			// 				"gravity": "center",
			// 			},
			// 			map[string]interface{}{
			// 				"type":    "text",
			// 				"text":    "CLICK!!",
			// 				"color":   "#ffffff",
			// 				"align":   "start",
			// 				"size":    "xs",
			// 				"gravity": "center",
			// 				"margin":  "lg",
			// 			},
			// 			map[string]interface{}{
			// 				"type":   "box",
			// 				"layout": "vertical",
			// 				"contents": []interface{}{
			// 					map[string]interface{}{
			// 						"type":            "box",
			// 						"layout":          "vertical",
			// 						"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
			// 						"width":           "70%",
			// 						"backgroundColor": "#0D8186",
			// 						"height":          "6px",
			// 					},
			// 				},
			// 				"backgroundColor": "#9FD8E36E",
			// 				"height":          "6px",
			// 				"margin":          "sm",
			// 			},
			// 		},
			// 		"backgroundColor": "#0000FF",
			// 		"paddingTop":      "19px",
			// 		"paddingAll":      "12px",
			// 		"paddingBottom":   "16px",
			// 	},
			// 	"action": map[string]interface{}{
			// 		"type":  "uri",
			// 		"label": "CREDIT",
			// 		"uri":   "https://liff.line.me/2006741358-z3P8WOYK",
			// 	},
			// 	"styles": map[string]interface{}{
			// 		"footer": map[string]interface{}{
			// 			"separator": false,
			// 		},
			// 	},
			// },
		},
	}

	return flex
}
func GenerateFlexHome() *FlexMessage {
	flex := &FlexMessage{
		Type: "carousel",
		Contents: []interface{}{
			// First Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "หลังบ้าน",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "30%",
									"backgroundColor": "#DE5658",
									"height":          "6px",
								},
							},
							"backgroundColor": "#FAD2A76E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#FF0000",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "รวมทุกอย่าง",
					"uri":   liffALL,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// map[string]interface{}{
			// 	"type": "bubble",
			// 	"size": "nano",
			// 	"header": map[string]interface{}{
			// 		"type":   "box",
			// 		"layout": "vertical",
			// 		"contents": []interface{}{
			// 			map[string]interface{}{
			// 				"type":    "text",
			// 				"text":    "Game",
			// 				"color":   "#ffffff",
			// 				"align":   "start",
			// 				"size":    "md",
			// 				"gravity": "center",
			// 			},
			// 			map[string]interface{}{
			// 				"type":    "text",
			// 				"text":    "CLICK!!",
			// 				"color":   "#ffffff",
			// 				"align":   "start",
			// 				"size":    "xs",
			// 				"gravity": "center",
			// 				"margin":  "lg",
			// 			},
			// 			map[string]interface{}{
			// 				"type":   "box",
			// 				"layout": "vertical",
			// 				"contents": []interface{}{
			// 					map[string]interface{}{
			// 						"type":            "box",
			// 						"layout":          "vertical",
			// 						"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
			// 						"width":           "70%",
			// 						"backgroundColor": "#0D8186",
			// 						"height":          "6px",
			// 					},
			// 				},
			// 				"backgroundColor": "#9FD8E36E",
			// 				"height":          "6px",
			// 				"margin":          "sm",
			// 			},
			// 		},
			// 		"backgroundColor": "#0000FF",
			// 		"paddingTop":      "19px",
			// 		"paddingAll":      "12px",
			// 		"paddingBottom":   "16px",
			// 	},
			// 	"action": map[string]interface{}{
			// 		"type":  "uri",
			// 		"label": "CREDIT",
			// 		"uri":   "https://liff.line.me/2006741358-z3P8WOYK",
			// 	},
			// 	"styles": map[string]interface{}{
			// 		"footer": map[string]interface{}{
			// 			"separator": false,
			// 		},
			// 	},
			// },
		},
	}

	return flex
}
func GenerateFlexHome2() *FlexMessage {
	flex := &FlexMessage{
		Type: "carousel",
		Contents: []interface{}{
			// First Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "RealTime",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "70%",
									"backgroundColor": "#0D8186",
									"height":          "6px",
								},
							},
							"backgroundColor": "#9FD8E36E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#0000FF",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "action",
					"uri":   liffRealTime,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// Second Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "เคลียร์ยอด",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "30%",
									"backgroundColor": "#DE5658",
									"height":          "6px",
								},
							},
							"backgroundColor": "#FAD2A76E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#FF0000",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "CLEAR",
					"uri":   liffClear,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			// Third Bubble
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "ผลรวมทุกคู่",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "70%",
									"backgroundColor": "#0D8186",
									"height":          "6px",
								},
							},
							"backgroundColor": "#9FD8E36E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#0000FF",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "CREDIT",
					"uri":   liffSummary,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "หลังบ้าน",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "30%",
									"backgroundColor": "#DE5658",
									"height":          "6px",
								},
							},
							"backgroundColor": "#FAD2A76E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#FF0000",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "HOME",
					"uri":   liffHQ,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
			map[string]interface{}{
				"type": "bubble",
				"size": "nano",
				"header": map[string]interface{}{
					"type":   "box",
					"layout": "vertical",
					"contents": []interface{}{
						map[string]interface{}{
							"type":    "text",
							"text":    "Game",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "md",
							"gravity": "center",
						},
						map[string]interface{}{
							"type":    "text",
							"text":    "CLICK!!",
							"color":   "#ffffff",
							"align":   "start",
							"size":    "xs",
							"gravity": "center",
							"margin":  "lg",
						},
						map[string]interface{}{
							"type":   "box",
							"layout": "vertical",
							"contents": []interface{}{
								map[string]interface{}{
									"type":            "box",
									"layout":          "vertical",
									"contents":        []interface{}{map[string]interface{}{"type": "filler"}},
									"width":           "70%",
									"backgroundColor": "#0D8186",
									"height":          "6px",
								},
							},
							"backgroundColor": "#9FD8E36E",
							"height":          "6px",
							"margin":          "sm",
						},
					},
					"backgroundColor": "#0000FF",
					"paddingTop":      "19px",
					"paddingAll":      "12px",
					"paddingBottom":   "16px",
				},
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "CREDIT",
					"uri":   liffGame,
				},
				"styles": map[string]interface{}{
					"footer": map[string]interface{}{
						"separator": false,
					},
				},
			},
		},
	}

	return flex
}
func getTextColor(value int) string {
	// Return a color based on the value (this can be customized further)
	if value > 0 {
		return "#00FF00" // Green for positive numbers
	}
	return "#FF0000" // Red for negative or zero
}
func summarize3(command string, round int) (string, error) {
	// Connect to the database
	dsn := "duckcom_fulloption2:duckcom_fulloption2@tcp(203.170.129.1:3306)/duckcom_fulloption2"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("Error: Failed to connect to the database.")
		return "", fmt.Errorf("failed to connect to the database: %v", err)
	}
	defer db.Close()
	log.Println("Debug: Database connection established.")

	// Query user data
	query := `
		SELECT ID, Name, Credit, Credit2, Number 
		FROM user_data 
		WHERE (Credit - Credit2) != ?
		ORDER BY (Credit - Credit2) DESC
	`
	rows, err := db.Query(query, 0)
	if err != nil {
		log.Println("Error: Failed to fetch user data.")
		return "", fmt.Errorf("failed to fetch user data: %v", err)
	}
	defer rows.Close()
	log.Println("Debug: User data query executed successfully.")

	// Process user data
	// Update struct to include Number
	userSums := []struct {
		ID     string
		Name   string
		Sum    float64
		Number int64 // ← add this field
	}{}

	var message strings.Builder

	for rows.Next() {
		var id sql.NullString
		var name sql.NullString
		var credit, credit2 sql.NullFloat64
		var number sql.NullInt64

		log.Println("Debug: Processing row from user data.")
		err := rows.Scan(&id, &name, &credit, &credit2, &number)
		if err != nil {
			log.Printf("Error: Failed to scan user data: %v", err)
			continue
		}

		log.Printf("Debug: Row data - ID: %v, Name: %v, Credit: %v, Credit2: %v, Number: %v",
			id, name, credit, credit2, number)

		if !id.Valid || !name.Valid || !credit.Valid || !credit2.Valid || !number.Valid {
			log.Println("Debug: Skipping row with NULL values.")
			continue
		}

		// Append user data to the slice
		userSums = append(userSums, struct {
			ID     string
			Name   string
			Sum    float64
			Number int64
		}{
			ID:     id.String,
			Name:   fmt.Sprintf("%d.%s", number.Int64, name.String),
			Sum:    credit.Float64 - credit2.Float64,
			Number: number.Int64, // ← store the number for sorting
		})

	}

	// Sort userSums by Sum in descending order
	// NEW: sort by Number descending
	sort.Slice(userSums, func(i, j int) bool {
		return userSums[i].Sum > userSums[j].Sum
	})

	// Update message with sorted user sums
	for _, user := range userSums {
		// Fetch user credits from the database
		var credit, credit2 float64
		err := db.QueryRow("SELECT Credit, Credit2 FROM user_data WHERE ID = ?", user.ID).Scan(&credit, &credit2)
		if err != nil {
			log.Printf("Error: Failed to fetch credits for user ID %s.", user.ID)
			return "", fmt.Errorf("failed to fetch credits for user %s: %v", user.ID, err)
		}
		log.Printf("Debug: User ID %s - Credit: %f, Credit2: %f", user.ID, credit, credit2)

		ball := credit - credit2
		formattedBall := formatWithCommas2(int(ball)) // Format the ball value (converted to int)

		// Prepare the message
		// for i := 0; i < 14; i++ {
		message.WriteString(fmt.Sprintf("%s//,//%s\n", user.Name, formattedBall))
		// }

		log.Printf("Debug: Message updated for user ID %s: %s", user.ID, message.String())
	}

	// Finalize message
	if message.Len() == 0 {
		log.Println("Debug: No user data processed. Returning default message.")
		return "----", nil
	}
	log.Println("Debug: Function completed successfully. Returning message.")
	return message.String(), nil
}

// Helper function to format numbers with commas
func formatWithCommas2(value int) string {
	return fmt.Sprintf("%s", strconv.FormatInt(int64(value), 10))
}

// Helper function to format numbers with commas

func formatNumber(num int) string {
	return fmt.Sprintf("%,d", num)
}
func createNewBubbleTemplate(backgroundURL string) Bubble {
	bubble := Bubble{
		Type: "bubble",
	}
	bubble.Body.Type = "box"
	bubble.Body.Contents = []interface{}{
		map[string]interface{}{
			"type":       "image",
			"url":        backgroundURL,
			"size":       "full",
			"aspectMode": "cover",
		},
		map[string]interface{}{
			"type":     "box",
			"layout":   "vertical",
			"contents": []interface{}{},
		},
	}
	return bubble
}

func generateFlexMessageX(data string, side int) *FlexMessage {
	// Determine background URL based on `side`
	var backgroundURL string
	switch side {
	case 1:
		backgroundURL = "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRSm4eK9C1LMOxb_snwPoWd-WQhMttXQq-KznQEQkRo68UpmGv3gSO_tKQ6x7JYVVoXQKo&usqp=CAU"
	case 0:
		backgroundURL = "https://static.vecteezy.com/system/resources/thumbnails/033/535/363/small/broken-glass-animation-green-screen-free-video.jpg"
	default:
		backgroundURL = "https://media.istockphoto.com/id/1130072614/vector/red-blur-bokeh-light-background.jpg?s=612x612&w=0&k=20&c=plR0D9M6vYR_2O7JfaMCqN0Ukwp_ijOlR_mMbdj2nug="
	}

	// Split input data into lines
	lines := strings.Split(data, "\n")
	var bubbles []Bubble
	currentBubble := createNewBubbleTemplate(backgroundURL)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse the line for player details
		parts := strings.Split(line, "//,//")
		if len(parts) < 2 {
			// If there are not enough parts, set default values
			parts = append(parts, "", "") // Add empty strings for missing data
		}
		namePart := parts[0]
		detailPart := parts[1]

		// Declare number and name variables
		var number, name string

		// Extract name and number
		nameStart := strings.Index(namePart, ")") + 1
		if nameStart <= 0 {
			// If nameStart is not found, set to default values
			number = "N/A"
			name = namePart
		} else {
			number = strings.Trim(namePart[1:nameStart-1], " ")
			name = strings.Trim(namePart[nameStart:], " ")
		}

		// Extract get and total values
		getTotal := strings.Split(detailPart, "=")
		if len(getTotal) < 2 {
			// If get and total are missing, set default values
			getTotal = append(getTotal, "0", "0")
		}
		get := strings.ReplaceAll(getTotal[0], ",", "")
		total := strings.ReplaceAll(getTotal[1], ",", "")
		getValue, _ := strconv.Atoi(get)
		totalValue, _ := strconv.Atoi(total)

		// Set color based on get value
		color := "#000000"
		if getValue >= 0 {
			color = "#00DA00"
		} else {
			color = "#FF0000"
		}

		// Add player information to current bubble
		playerBox := Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Box{
					Type:   "text",
					Text:   fmt.Sprintf("(%s)%s", number, name),
					Weight: "bold",
					// Flex:   2,
					Size: "sm",
				},
				Box{
					Type:  "text",
					Text:  fmt.Sprintf("%d", totalValue),
					Align: "end",
					// Flex:   2,
					Weight: "bold",
					Color:  color,
					//Wrap:   true,
					Size: "sm",
				},
			},
			Margin: "sm",
		}
		currentBubble.Body.Contents = append(currentBubble.Body.Contents, playerBox)

		// Add separator
		currentBubble.Body.Contents = append(currentBubble.Body.Contents, Box{
			Type:            "box",
			Layout:          "horizontal",
			Contents:        []interface{}{},
			Height:          "1px",
			BackgroundColor: "#AAAAAA",
			Margin:          "sm",
		})

		// After every 80 entries, finalize the current bubble and start a new one
		if (i+1)%80 == 0 || (i+1) == len(lines) {
			bubbles = append(bubbles, currentBubble)
			currentBubble = createNewBubbleTemplate(backgroundURL)
		}
	}

	// Add the last bubble if not empty
	if len(currentBubble.Body.Contents) > 0 {
		bubbles = append(bubbles, currentBubble)
	}

	// Create FlexMessage carousel
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: []interface{}{},
	}

	// Append all bubbles
	for _, bubble := range bubbles {
		flexMessage.Contents = append(flexMessage.Contents, bubble)
	}

	return &flexMessage
}

func IllustrateU(dep, profit, redWin, blueWin int64) *FlexMessage {

	// Use the default image URL if pictureURL is empty
	pictureURL := "https://w7.pngwing.com/pngs/1000/665/png-transparent-computer-icons-profile-s-free-angle-sphere-profile-cliparts-free-thumbnail.png"
	//depColor, _ := strconv.Atoi(dep)
	proColor := int(profit)
	redColor := int(redWin)
	blueColor := int(blueWin)
	loc, _ := time.LoadLocation("Asia/Bangkok")
	now := time.Now().In(loc)

	// สร้าง Printer สำหรับภาษาไทย
	p := message.NewPrinter(language.Thai)

	// กำหนดชื่อเดือนเป็นภาษาไทย
	// months := []string{
	// 	"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
	// 	"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	// }

	// จัดรูปแบบวันที่เป็นภาษาไทย
	year := number.Decimal(now.Year()+543, number.NoSeparator())
	// thaiDate := p.Sprintf("%d %s %d", now.Day(), months[int(now.Month())], now.Year()+543)

	// กำหนดชื่อเดือนเป็นภาษาไทย
	// months := []string{
	// 	"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
	// 	"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	// }

	// จัดรูปแบบปี พ.ศ. โดยไม่มีตัวคั่นหลักพัน

	// จัดรูปแบบวันที่เป็นภาษาไทย
	thaiDate := p.Sprintf("%d/%d/%v", now.Day(), int(now.Month()), year)
	// Header section with user details and profile image
	headerContents := Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			Box{
				Type:   "box",
				Layout: "vertical",

				Contents: []interface{}{
					Image{
						Type:       "image",
						Url:        pictureURL, // Default image URL
						OffsetEnd:  "0px",
						AspectMode: "cover",
					},
				},

				Width:   "00px",
				Spacing: "none",
				Margin:  "md",
			},
			// Box{
			// 	Type:   "box",
			// 	Layout: "horizontal",
			// 	Contents: []interface{}{
			// 		Text{
			// 			Type:  "text",
			// 			Text:  thaiDate,
			// 			Size:  "xxl",
			// 			Align: "center",
			// 			Color: "#FFFFFF",
			// 		},
			// 	},
			// },
		},
		BackgroundColor: "#222222",
		// Height:          "33px",
	}

	// Body section with dynamic content
	bodyContents := []interface{}{
		Box{
			Type:   "box",
			Layout: "horizontal",

			Contents: []interface{}{

				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#00ff00", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   "ยอดฝาก",
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  "#000000",
								},
							},
							// Simulating border by using padding and margin

						},
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#000000", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   "ได้/เสีย วันนี้",
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  "#ffffff",
								},
							},
							// Simulating border by using padding and margin

						},
					},
					//  Margin: "sm",
				},

				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#00ff00", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   formatNumberWithCommas(int64(dep)),
									Size:   "lg",
									Weight: "bold",
									Align:  "center",

									Color: "#000000",
								},
							},
							// Simulating border by using padding and margin

						},
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#000000", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   formatNumberWithCommas(int64(profit)),
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  getColor22(proColor),
								},
							},
							// Simulating border by using padding and margin

						},
					},
					// Margin: "sm",
				},
			},
			// Outer margin or padding for the box itself
			// Margin: "sm",
		},
		Box{
			Type:   "box",
			Layout: "vertical",
			Contents: []interface{}{
				Box{
					Type:            "box",
					Layout:          "vertical",
					BackgroundColor: "#ffffff", //t the background color
					// Height:          "75px",
					JustifyContent: "center",
					Contents: []interface{}{
						Text{
							Type:   "text",
							Text:   formatNumberWithCommas(int64(dep)),
							Size:   "lg",
							Weight: "bold",
							Align:  "center",

							Color: "#ffffff",
						},
					},
					// Simulating border by using padding and margin

				},
			},
			// Margin: "sm",
		},
		Box{
			Type:   "box",
			Layout: "horizontal",

			Contents: []interface{}{
				Box{
					Type:   "box",
					Layout: "horizontal",
					Contents: []interface{}{
						Text{
							Type:   "text",
							Text:   thaiDate,
							Size:   "xxl",
							Align:  "center",
							Weight: "bold",
							Color:  "#000000",
						},
					},
				},

				// Outer margin or padding for the box itself
				// Margin: "sm",
			}},
		Box{
			Type:   "box",
			Layout: "horizontal",

			Contents: []interface{}{

				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#ff0000", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   "แดงชนะ",
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  "#ffffff",
								},
							},
							// Simulating border by using padding and margin

						},
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#000000", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   formatNumberWithCommas(int64(redWin)),
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  getColor22(redColor),
								},
							},
							// Simulating border by using padding and margin

						},
					},
					// Margin: "sm",
				},
				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#0000ff", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   "น้ำเงินชนะ",
									Size:   "lg",
									Weight: "bold",
									Align:  "center",

									Color: "#ffffff",
								},
							},
							// Simulating border by using padding and margin

						},
						Box{
							Type:            "box",
							Layout:          "vertical",
							BackgroundColor: "#000000", //t the background color
							Height:          "33px",
							JustifyContent:  "center",
							Contents: []interface{}{
								Text{
									Type:   "text",
									Text:   formatNumberWithCommas(int64(blueWin)),
									Size:   "lg",
									Weight: "bold",
									Align:  "center",
									Color:  getColor22(blueColor),
								},
							},
							// Simulating border by using padding and margin

						},
					},
					// Margin: "sm",
				},
			},
			// Outer margin or padding for the box itself
			Margin: "xxl",
		},
	}

	// Footer section
	footerContents := []interface{}{
		Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Text{
					Type:  "text",
					Text:  houseName,
					Size:  "xs",
					Align: "center",
					Color: "#888888",
				},
			},
		},
	}

	// Create the bubble
	bubble := Bubble{
		Type:   "bubble",
		Size:   "mega",
		Hero:   headerContents,
		Body:   Box{Type: "box", Layout: "vertical", Contents: bodyContents},   // Wrap bodyContents inside a Box
		Footer: Box{Type: "box", Layout: "vertical", Contents: footerContents}, // Wrap footerContents inside a Box
	}

	// Create the carousel
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: []interface{}{bubble},
	}

	return &flexMessage
}
func GenerateFlexCancelMessage(message string) (*FlexMessage, error) {
	// Header section with dynamic content
	headerContents := Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			Text{
				Type:         "text",
				Text:         "ยกเลิกเดิมพันรอบล่าสุด", // Static title text
				Color:        "#ffffff",
				Align:        "start",
				Gravity:      "center",
				OffsetBottom: "100px",
				OffsetStart:  "120px",
			},
			Text{
				Type:         "text",
				Text:         message, // Dynamic cancellation message
				Color:        "#ffffff",
				Align:        "start",
				Gravity:      "center",
				OffsetBottom: "100px",
				OffsetStart:  "120px",
			},
		},
		BackgroundColor: "#222222",
		Height:          "100px",
	}

	// Body section (empty in this case as it's just a cancellation message)
	bodyContents := []interface{}{
		Box{
			Type:   "box",
			Layout: "vertical",
			Contents: []interface{}{
				Box{
					Type: "separator", // Separator between header and footer
				},
			},
		},
	}

	// Footer section
	footerContents := []interface{}{
		Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Text{
					Type:  "text",
					Text:  houseName,
					Size:  "xs",
					Align: "center",
					Color: "#888888",
				},
			},
		},
	}

	// Create the bubble
	bubble := Bubble{
		Type:   "bubble",
		Size:   "kilo",
		Hero:   headerContents,
		Body:   Box{Type: "box", Layout: "vertical", Contents: bodyContents},   // Wrap bodyContents inside a Box
		Footer: Box{Type: "box", Layout: "vertical", Contents: footerContents}, // Wrap footerContents inside a Box
	}

	// Create the Flex message
	flexMessage := FlexMessage{
		Type:     "bubble",
		Contents: []interface{}{bubble},
	}

	return &flexMessage, nil
}

func GenerateFlexC(userID string, userName string, credit string, credit2 string, redWin string, blueWin string, round string, pictureURL string, redBalance string, blueBalance string, redBalance2 string, blueBalance2 string) (*FlexMessage, error) {
	if pictureURL == "" {
		// Use the default image URL if pictureURL is empty
		pictureURL = "https://cdn-icons-png.flaticon.com/512/5734/5734068.png"
	}
	// Header section with user details and profile image
	c1Color, _ := strconv.Atoi(credit)
	c2Color, _ := strconv.Atoi(credit2)
	redColor, _ := strconv.Atoi(redWin)
	blueColor, _ := strconv.Atoi(blueWin)
	redColor2, _ := strconv.Atoi(redBalance2)
	blueColor2, _ := strconv.Atoi(blueBalance2)
	// redColor3, _ := strconv.Atoi(redBalance)
	// blueColor3, _ := strconv.Atoi(blueBalance)
	headerContents := Box{
		Type:   "box",
		Layout: "horizontal",
		Contents: []interface{}{
			Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type:   "box",
						Layout: "vertical",
						Contents: []interface{}{
							Image{
								Type:       "image",
								Url:        pictureURL,
								AspectMode: "cover",
							},
						},
						Width:        "60px",
						BorderColor:  "#FFD700", // สีทอง
						BorderWidth:  "2px",     // ความหนาเส้นขอบ
						CornerRadius: "sm",
						Margin:       "md",
					},
					Text{
						Type:    "text",
						Text:    "(" + userID + ")" + userName, // Display user name
						Color:   "#ffffff",
						Align:   "start",
						Size:    "xxs",
						Gravity: "center",
					},
				},

				// BorderColor:  "#FFD700", // สีทอง
				JustifyContent: "center",
				Width:          "70px",
				Margin:         "md",
			},
			Box{
				Type:   "box",
				Layout: "horizontal",
				Contents: []interface{}{
					Box{
						Type:   "box",
						Layout: "vertical",
						Contents: []interface{}{
							Text{
								Type:    "text",
								Text:    "💳เครดิต: ",
								Weight:  "bold",
								Align:   "start",
								Gravity: "center",
								Color:   "#FFD700", // สีทอง
								Size:    "xxs",
							},
							Text{
								Type:    "text",
								Text:    "💸คงเหลือ: ",
								Size:    "xxs",
								Align:   "start",
								Gravity: "center",
								Color:   "#1DB446",
								Weight:  "bold",
							},

							Text{
								Type:   "text",
								Text:   "แดงชนะ",
								Size:   "xxs",
								Weight: "bold",
								Align:  "start",
								Color:  "#FF0000",
							},
							Box{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: "#ff0000", //t the background color
								Contents: []interface{}{
									Text{
										Type:   "text",
										Text:   formatNumberWithCommas(int64(redColor)),
										Size:   "sm",
										Weight: "bold",
										Align:  "center",
										Color:  "#ffffff",
									},
								},
								// Simulating border by using padding and margin
								BorderColor:  "#FFD700", // สีทอง
								BorderWidth:  "1px",     // ความหนาเส้นขอบ
								CornerRadius: "sm",
								Width:        "88%",
							},
							Text{
								Type:   "text",
								Text:   "คงเหลือ",
								Size:   "xxs",
								Weight: "bold",
								Align:  "start",
								Color:  "#FF0000",
							},
							Box{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: "#ff0000", //t the background color
								Contents: []interface{}{
									Text{
										Type:   "text",
										Text:   formatNumberWithCommas(int64(redColor2)),
										Size:   "sm",
										Weight: "bold",
										Align:  "center",
										Color:  "#ffffff",
									},
								},
								// Simulating border by using padding and margin
								BorderColor:  "#ffffff", // สีทอง
								BorderWidth:  "1px",     // ความหนาเส้นขอบ
								CornerRadius: "sm",
								Width:        "88%",
							},
						},
						Margin: "sm",
					},
					Box{
						Type:   "box",
						Layout: "vertical",
						Contents: []interface{}{
							Text{
								Type:    "text",
								Text:    formatNumberWithCommas(int64(c2Color)) + "฿",
								Weight:  "bold",
								Align:   "start",
								Gravity: "center",
								Color:   "#FFD700", // สีทอง
								Size:    "xxs",
							},
							Text{
								Type:    "text",
								Text:    formatNumberWithCommas(int64(c1Color)) + "฿",
								Size:    "xxs",
								Align:   "start",
								Gravity: "center",
								Color:   "#1DB446",
								Weight:  "bold",
							},

							Text{
								Type:   "text",
								Text:   "น้ำเงินชนะ",
								Size:   "xxs",
								Weight: "bold",
								Align:  "start",
								Color:  "#20a7db",
							},
							Box{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: "#0000ff", //t the background color
								Contents: []interface{}{
									Text{
										Type:   "text",
										Text:   formatNumberWithCommas(int64(blueColor)),
										Size:   "sm",
										Weight: "bold",
										Align:  "center",
										Color:  "#ffffff",
									},
								},
								// Simulating border by using padding and margin
								CornerRadius: "sm",
								BorderColor:  "#FFD700", // สีทอง
								BorderWidth:  "1px",     // ความหนาเส้นขอบ

								Width: "88%",
							},
							Text{
								Type:   "text",
								Text:   "คงเหลือ",
								Size:   "xxs",
								Weight: "bold",
								Align:  "start",
								Color:  "#20a7db",
							},
							Box{
								Type:            "box",
								Layout:          "vertical",
								BackgroundColor: "#0000ff", //t the background color
								Contents: []interface{}{
									Text{
										Type:   "text",
										Text:   formatNumberWithCommas(int64(blueColor2)),
										Size:   "sm",
										Weight: "bold",
										Align:  "center",
										Color:  "#ffffff",
									},
								},
								// Simulating border by using padding and margin
								CornerRadius: "sm",
								BorderColor:  "#ffffff", // สีทอง
								BorderWidth:  "1px",     // ความหนาเส้นขอบ

								Width: "88%",
							},
						},
						Margin: "sm",
					},
				},
				JustifyContent: "space-around",
				Margin:         "sm",
			},
		},
		BackgroundColor: "#222222",
		Height:          "133px",
	}

	// Body section with dynamic content
	bodyContents := []interface{}{

		Box{
			Type:   "box",
			Layout: "vertical", // Corrected layout to "vertical"
			// Set the height of the box
			Contents: []interface{}{
				Box{
					Type: "separator", // Separator between the elements
				},
			},
			Height: "00px",
			Width:  "00px",
		},
	}

	// Footer section
	actionP := Action{
		Type: "uri",
		URI:  PlayRoom,
	}
	actionD := Action{
		Type: "uri",
		URI:  DepositRoom,
	}
	// Footer section
	footerContents := []interface{}{Box{
		Type:    "box",
		Layout:  "horizontal",
		Spacing: "sm",
		Contents: []interface{}{
			Box{
				Type:            "box",
				Layout:          "vertical",
				CornerRadius:    "sm",
				BorderWidth:     "light",
				BorderColor:     "#ffc60a",
				BackgroundColor: "#ff0000",

				Contents: []interface{}{
					Text{
						Type:  "text",
						Text:  "ไปห้องเล่น",
						Size:  "xxs",
						Color: "#ffffff",
						Align: "center",
					},
				},
				Action: &actionP,
			},
			Box{
				Type:            "box",
				Layout:          "vertical",
				CornerRadius:    "sm",
				BorderWidth:     "light",
				BorderColor:     "#ffc60a",
				BackgroundColor: "#00c000",

				Contents: []interface{}{
					Text{
						Type:  "text",
						Text:  "ไปห้องฝาก",
						Size:  "xxs",
						Color: "#ffffff",
						Align: "center",
					},
				},
				Action: &actionD,
			},
		},
	}}

	// Create the bubble
	bubble := Bubble{
		Type:   "bubble",
		Size:   "kilo",
		Hero:   Box{Type: "box", Layout: "vertical", Contents: bodyContents, Height: "0px"},
		Body:   headerContents,                                                                             // Wrap bodyContents inside a Box
		Footer: Box{Type: "box", Layout: "vertical", Contents: footerContents, BackgroundColor: "#000000"}, // Wrap footerContents inside a Box
	}

	// Create the carousel
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: []interface{}{bubble},
	}

	return &flexMessage, nil
}
func GenerateFlexMessageC2(te [][]string, userName string, userPic string, userID string, redBal string, blueBal string) (*FlexMessage, error) {
	// Initialize variables
	var flexMessages []interface{}
	var contents []interface{}
	var contentsLeft []interface{}
	var contentsRight []interface{}
	if userPic == "" {
		userPic = "https://i.postimg.cc/43y8KCQ5/Pngtree-incognito-8649979.png"
	}
	rowCount := 0
	maxRows := 100 // Limit the rows to
	money1 := getBalance(userID)
	contents = append(contents, Box{
		Type:   "box",
		Layout: "vertical",
		Contents: []interface{}{
			Box{
				Type: "separator",
			},
		},
	})

	// Iterate through the 2D array to create dynamic rows based on input
	for _, row := range te[1 : len(te)-1] { // Skip the header row
		if rowCount >= maxRows { // Stop processing if we reach 100 rows
			break
		}
		// var formGet, formRemain string
		// If data is empty, replace it with "----"
		name := row[0]
		if name == "" {
			name = "----"
		}

		get := row[1]
		if get == "" {
			get = "----"
		}

		// Add row to contents
		// Determine background color based on the first character of "get"
		backgroundColor := "#FF0000"                                      // Default white background
		fmt.Println(get, len([]rune(get)), string([]rune(get)[1]), "TVS") // Debugging output
		if len([]rune(get)) > 0 {
			// Use the first character in `get` as a rune
			firstChar := string([]rune(get)[1])
			if firstChar == "ด" {
				backgroundColor = "#FF0000" // Red for "ด"
				contentsLeft = append(contentsLeft, Box{
					Type:   "box",
					Layout: "vertical",
					// Height:  "52px",
					// PaddingAll: "0px",
					// Spacing: "none",

					// BackgroundColor: "#eeeeee",

					BackgroundColor: backgroundColor, // Use the pre-calculated color
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   fmt.Sprintf("%s%s", name, get), // Combine name and get into one text
							Weight: "bold",
							Size:   "10px",
							Align:  "end",     // Center-align the text
							Color:  "#FFFFFF", // Text color will always be white
						},
					},
				})

				contentsLeft = append(contentsLeft, Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type: "separator",
						},
					},
				})
			} else if firstChar == "ง" {
				backgroundColor = "#0000FF" // Blue for "ง"
				contentsRight = append(contentsRight, Box{
					Type:   "box",
					Layout: "vertical",
					// Height:  "52px",
					// PaddingAll: "0px",
					Spacing: "xs",

					// BackgroundColor: "#eeeeee",

					BackgroundColor: backgroundColor, // Use the pre-calculated color
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   fmt.Sprintf("%s%s", name, get), // Combine name and get into one text
							Weight: "bold",
							Size:   "10px",
							Align:  "end",     // Center-align the text
							Color:  "#FFFFFF", // Text color will always be white
						},
					},
				})

				contentsRight = append(contentsRight, Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type: "separator",
						},
					},
				})
			}
		}

		rowCount++

		if rowCount == maxRows {
			round, sub, _, _, _, _, _, _, _, _, _, _ := GetLocalVar()
			hero := Box{
				Type:    "box",
				Layout:  "vertical",
				Height:  "30px",
				Spacing: "xl",
				Contents: []interface{}{
					Box{
						Type:   "text",
						Text:   fmt.Sprintf("สรุปการเดิมพัน# %d (%d)", round, sub),
						Weight: "bold",
						Size:   "xxs",
						Align:  "center",
					},
					Box{
						Type: "separator",
					},
				},
			}

			body := Box{
				Type:     "box",
				Layout:   "vertical",
				Contents: contents, // Add dynamically created contents
			}
			footer := Box{
				Type:   "box",
				Layout: "horizontal",
				Contents: []interface{}{
					Box{
						Type:  "text",
						Text:  houseName,
						Size:  "xs",
						Align: "center",  // Footer text centered
						Color: "#888888", // Dark gray color
						//Action:          &uriAction,
						BackgroundColor: "",
					},
				},
			}

			// Create bubble containing hero, body, and footer
			bubble := Bubble{
				Type:   "bubble",
				Size:   "giga",
				Hero:   hero,
				Body:   body,
				Footer: footer,
			}

			// Add the bubble to the carousel
			flexMessages = append(flexMessages, bubble)

			// Reset contents for the next bubble
			contents = []interface{}{
				Box{
					Type:   "box",
					Layout: "horizontal",
					Contents: []interface{}{
						Box{
							Type:   "text",
							Text:   "ผู้เล่น",
							Weight: "bold",
							Size:   "sm",
							Align:  "start",
						},
						Box{
							Type:   "text",
							Text:   "การเดิมพัน",
							Weight: "bold",
							Size:   "sm",
							Align:  "end",
							Color:  "#FF0000",
						},
					},
				},
				Box{
					Type:   "box",
					Layout: "vertical",
					Contents: []interface{}{
						Box{
							Type: "separator",
						},
					},
				},
			}
			rowCount = 0 // Reset row count for the next bubble
		}
	}

	// After the loop, if there are any remaining rows that weren't added to a bubble, create a final bubble
	if rowCount >= 0 {
		round, _, _, _, _, _, _, _, _, _, _, _ := GetLocalVar()
		hero := Box{
			Type:   "box",
			Layout: "horizontal", // Horizontal layout for side-by-side elements
			// Height:  "30px",
			Spacing: "xs",
			Contents: []interface{}{
				// Profile picture box (1/4 of space)
				Box{
					Type:  "image",
					Align: "start",
					// BorderWidth: "3px",

					URL:  userPic, // Replace with the profile image URL
					Size: "xxs",

					Flex: 1, // Allocate 1/4 of the space
				},
				// Content box (3/4 of space)

				Box{
					Type:    "box",
					Layout:  "vertical",
					Spacing: "xs",
					Flex:    12, // Allocate 3/4 of the space
					Contents: []interface{}{
						Box{
							Type:   "box",
							Layout: "horizontal", // Horizontal layout for side-by-side elements
							// Height:  "30px",
							Spacing: "xs",
							Contents: []interface{}{
								Box{
									Type:   "text",
									Text:   userName,
									Weight: "bold",
									Size:   "sm",
									Align:  "start",
									Flex:   5,
								},
								Box{
									Type:   "text",
									Text:   fmt.Sprintf("คู่ที่ %d แดง", round),
									Weight: "bold",
									Size:   "sm",
									Align:  "start",
									Color:  "#000000",
									Flex:   4,
								},
								Box{
									Type:   "text",
									Text:   fmt.Sprintf("น้ำเงิน"),
									Weight: "bold",
									Size:   "sm",
									Align:  "center",
									Color:  "#000000",
									Flex:   4,
								},
							},
						},

						Box{
							Type:   "box",
							Layout: "horizontal", // Horizontal layout for side-by-side elements
							// Height:  "30px",
							Spacing: "xs",
							Contents: []interface{}{
								Box{
									Type:   "box",
									Layout: "vertical",
									// Height:  "52px",
									Spacing: "xs",
									// BackgroundColor: "#eeeeee",
									Flex: 14,
									// BorderWidth:     "xxs",
									BackgroundColor: "#22cc22", // Use the pre-calculated color
									Contents: []interface{}{
										Box{

											Type:   "text",
											Text:   fmt.Sprintf("ทุน:%v", formatWithCommas(money1)),
											Weight: "bold",
											Size:   "xs",
											Align:  "center",
											Color:  "#ffffff",

											Flex: 16,
										},
									},
								},

								Box{
									Type:   "box",
									Layout: "vertical",
									// Height:  "52px",
									Spacing: "xs",
									// BackgroundColor: "#eeeeee",
									Flex: 12,
									// BorderWidth:     "xxs",
									BackgroundColor: "#ff0000", // Use the pre-calculated color
									Contents: []interface{}{
										Box{

											Type:   "text",
											Text:   fmt.Sprintf("%v", redBal),
											Weight: "bold",
											Size:   "xs",
											Align:  "center",
											Color:  "#ffffff",

											Flex: 12,
										},
									},
								}, Box{
									Type:   "box",
									Layout: "vertical",
									// Height:  "52px",
									Spacing: "xs",
									// BackgroundColor: "#eeeeee",
									Flex: 12,
									// BorderWidth:     "xxs",
									BackgroundColor: "#0000ff", // Use the pre-calculated color
									Contents: []interface{}{
										Box{

											Type:   "text",
											Text:   fmt.Sprintf(" %v", blueBal),
											Weight: "bold",
											Size:   "xs",
											Align:  "center",
											Color:  "#ffffff",
											Flex:   12,
										},
									},
								},
								// , Box{

								// 	Type:   "text",
								// 	Text:   fmt.Sprintf("💳:%v", formatWithCommas(money2)),
								// 	Weight: "bold",
								// 	Size:   "sm",
								// 	Align:  "start",
								// 	Color:  "#aa8800",
								// },

							},
						},

						// Box{

						// 	Type:   "text",
						// 	Text:   fmt.Sprintf("ทุน:%v เครดิต:%v", money1, money2),
						// 	Weight: "bold",
						// 	Size:   "sm",
						// 	Align:  "start",
						// 	Color:  "#0000FF",
						// },
					},
				},
			},
		}
		if len(contentsLeft) < 1 {
			contentsLeft = append(contentsLeft, Box{
				Type:   "separator",
				Margin: "none", // No margin to keep it tight
			})

		}
		if len(contentsRight) < 1 {
			contentsRight = append(contentsRight, Box{
				Type:   "separator",
				Margin: "none", // No margin to keep it tight
			})

		}
		contents = append(contents, Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{
				Box{
					Type:     "box",
					Layout:   "vertical",
					Contents: contentsLeft, // Vertical box for the left side
					// Spacing:  "none",       // Ensure no padding (depending on your framework's Box structure)
				},
				// Add a separator between the left and right vertical boxes
				Box{
					Type:   "separator",
					Margin: "none", // No margin to make it tight
				},
				Box{
					Type:     "box",
					Layout:   "vertical",
					Contents: contentsRight, // Vertical box for the right side
					// Spacing:  "none",        // Ensure no padding
				},
			},
		})

		// contentsLeft = append(contents, Box{
		// 	Type:   "box",
		// 	Layout: "vertical",
		// 	Contents: []interface{}{
		// 		Box{
		// 			Type: "separator",
		// 		},
		// 	},
		// })

		body := Box{
			Type:     "box",
			Layout:   "vertical",
			Contents: contents,
		}

		// uriAction := Action{
		// 	Type: "uri",
		// 	URI:  liffProfile, // LIFF URL
		// }
		footer := Box{
			Type:   "box",
			Layout: "horizontal",
			Contents: []interface{}{

				Box{
					Type:  "text",
					Text:  houseName,
					Size:  "xxs",
					Align: "center",  // Footer text centered
					Color: "#888888", // Dark gray color
					//Action:          &uriAction,
					BackgroundColor: "",
					// Height:          "010x",
				},
			},
		}

		bubble := Bubble{
			Type:   "bubble",
			Size:   "giga",
			Hero:   hero,
			Body:   body,
			Footer: footer,
		}

		flexMessages = append(flexMessages, bubble)
	}

	// If no messages were created, provide a fallback message
	if len(flexMessages) == 0 {
		flexMessages = append(flexMessages, Bubble{
			Type: "bubble",
			Size: "giga",
			Body: Box{
				Type:   "box",
				Layout: "vertical",
				Contents: []interface{}{
					Box{
						Type:   "text",
						Text:   "ไม่มีข้อมูล",
						Weight: "bold",
						Size:   "xl",
						Align:  "center",
					},
				},
			},
		})
	}

	// Create final flex message with carousel layout
	flexMessage := FlexMessage{
		Type:     "carousel",
		Contents: flexMessages,
	}

	return &flexMessage, nil
}
func SummarizeC2(UserID string, UserName string) [][]string {
	// Fetch local variables
	localRound, localSub, localState, localRedRate, localBlueRate, _, _, localCommand, localMin, localMax, localWin, err := GetLocalVar()
	if err != nil {
		log.Printf("Error getting local variables: %v", err)
		return nil
	}

	// Log local variables
	log.Printf("Local Variables: Round=%d, Sub=%d, State=%d, RedRate=%.2f, BlueRate=%.2f, Command=%s, Min=%d, Max=%d, Win=%d",
		localRound, localSub, localState, localRedRate, localBlueRate, localCommand, localMin, localMax, localWin)

	// Query to fetch playing logs
	query := `
		SELECT b1, j1, red_rate, blue_rate, balance, Time2, win 
		FROM playinglog
		WHERE ID = ? AND round = ? AND Game_play != "END"
		ORDER BY Time DESC
	`

	// Initialize variables for query results
	var b1, j1 float64
	var redRate, blueRate, wantedTime string
	var balance, wantedWin int
	var wantedRate, amount string
	var sideHead = "ด"
	found := false

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute the query
	rows, err := db.QueryContext(ctx, query, UserID, localRound)
	if err != nil {
		log.Printf("Error executing query: %v\n", err)
		return nil
	}
	defer rows.Close()

	// Loop through rows and process data
	for rows.Next() {
		err := rows.Scan(&b1, &j1, &redRate, &blueRate, &balance, &wantedTime, &wantedWin)
		if err != nil {
			log.Printf("Error scanning row: %v\n", err)
		}
		sideHead = "ด"

		// Determine the side head and adjust redRate if needed
		if j1 > b1 {
			b1 = j1
			sideHead = "ง"
			redRate = blueRate
		}

		// Set wantedRate based on side head and win condition
		if sideHead == "ด" {
			if wantedWin == 1 {
				wantedRate = "รอง"
			} else {
				wantedRate = "ต่อ"
			}
		} else {
			if wantedWin == -1 {
				wantedRate = "รอง"
			} else {
				wantedRate = "ต่อ"
			}
		}

		// Format the time to "HH:mm"
		t, err := time.Parse("2006-01-02 15:04:05.999", wantedTime)
		wantedTime = t.Format("15:04")

		redRate2, _ := strconv.ParseFloat(redRate, 64)
		demo := 1

		numerator, denominator := floatToFraction(redRate2)
		demoNumerator := numerator * demo
		demoDenominator := denominator

		gcd := findGCD(demoNumerator, demoDenominator)
		demoNumerator /= gcd
		demoDenominator /= gcd

		amount += sideHead + formatWithCommas2(int(b1)) +
			fmt.Sprintf(" (%v %d/%d)  เมื่อ: (%s) \n", wantedRate, demoNumerator, demoDenominator, wantedTime)

		if j1 == b1 && j1 == 0 {
			amount = "ไม่มีแผลในระบบ"
		}

		fmt.Printf("Fetched row: b1=%f, j1=%f, red_rate=%d/%d, blue_rate=%s, balance=%d\n",
			b1, j1, demoNumerator, demoDenominator, blueRate, balance)
		fmt.Printf("Amount: %s\n", amount)

		found = true
	}

	// Check if no rows were found
	if !found {
		log.Println("No rows found for the specified username.")
		amount = "ไม่มีแผลในระบบ"
	} else {
		amount += fmt.Sprintf(" รวม : %v ฿ ", balance)
	}

	// Build and format the final message
	var messageBuilder strings.Builder
	messageBuilder.WriteString("ผู้เล่น//,// ได้เสีย  คงเหลือ\n")
	amountLines := strings.Split(strings.TrimSpace(amount), "\n")

	// Iterate over userOrder to build the final output
	if len(amountLines) > 0 {
		for i, line := range amountLines {
			if i == 0 {
				// For the first line, use UserName
				messageBuilder.WriteString(fmt.Sprintf(" //,// %s\n", line))
			} else {
				// For subsequent lines, use "..."
				messageBuilder.WriteString(fmt.Sprintf(" //,// %s\n", line))
			}
		}
	}

	// Return formatted output
	if messageBuilder.Len() == 0 {
		return [][]string{}
	}
	return splitAndFormatMessage(messageBuilder.String())
}
func transformRate2(rate string, wantedRate string) (string, error) {
	// Split the rate string into numerator and denominator
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid rate format: %s", rate)
	}

	// Parse the numerator and denominator as floats
	numerator, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "", fmt.Errorf("invalid numerator: %s", parts[0])
	}

	denominator, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return "", fmt.Errorf("invalid denominator: %s", parts[1])
	}

	// Special handling for cases where the numerator is 0.9 or 1
	if numerator == 0.9 {
		return "9/9", nil
	} else if numerator == 1 {
		return "10/10", nil
	}

	// Scale the numerator to 1 and adjust the denominator accordingly
	if numerator != 1 && denominator > 10 {
		denominator = denominator / numerator
		numerator = 1
	} else {
		denominator *= 10
		numerator *= 10
	}

	// Helper function to format a float as an integer if it has no decimal value
	formatNumber := func(num float64) string {
		if num == float64(int(num)) {
			return fmt.Sprintf("%d", int(num)) // Format as integer
		}
		return fmt.Sprintf("%.1f", num) // Format with one decimal point
	}

	// Reduce the fraction if the denominator is greater than 10
	if denominator > 10 {
		gcd := func(a, b float64) float64 {
			for b != 0 {
				a, b = b, math.Mod(a, b)
			}
			return a
		}
		d := gcd(numerator, denominator)
		if d > 1 {
			numerator /= d
			denominator /= d
		}
	}

	// Format numerator and denominator
	formattedNumerator := formatNumber(numerator)
	formattedDenominator := formatNumber(denominator)
	transformedRate := fmt.Sprintf("%s/%s", formattedNumerator, formattedDenominator)

	// Return the transformed rate
	if wantedRate == "รอง" {
		transformedRate = fmt.Sprintf("%s/%s", formattedDenominator, formattedNumerator)
	}

	return transformedRate, nil
}

// SendFlexMessage sends a Flex message to the LINE Messaging API.
func SendFlexMessage(replyToken string, flexMessage *FlexMessage, channelToken string) error {
	endpoint := "https://api.line.me/v2/bot/message/reply"

	message := map[string]interface{}{
		"replyToken": replyToken,
		"messages": []map[string]interface{}{
			{
				"type":     "flex",
				"altText":  "เช็คยอด",
				"contents": flexMessage,
			},
		},
	}

	jsonBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// fmt.Println("Request Payload:", string(jsonBody)) // Debugging print

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+channelToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
func CheckAdmin(UserID string) (int, error) {
	var adminStatus int
	ctx := context.Background()

	// Use parameterized query to avoid SQL injection
	query := `SELECT admin FROM user_data WHERE id = ? LIMIT 1`
	err := db.QueryRowContext(ctx, query, UserID).Scan(&adminStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows found, handle it appropriately (return 0 or an error)
			return 0, fmt.Errorf("no user found with ID: %s", UserID)
		}
		return 0, fmt.Errorf("failed to execute query: %v", err)
	}

	// Successfully retrieved the admin status
	return adminStatus, nil
}
