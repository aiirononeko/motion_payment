// payment パッケージはレシートの検証メソッドを提供します.
package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	firebase "firebase.google.com/go/v4"

	"google.golang.org/api/option"
)

// SANDBOX_URL はsandbox環境のURLです.
const SANDBOX_URL string = "https://sandbox.itunes.apple.com/verifyReceipt"

// PRODUCTION_URL は本番環境のURLです.
const PRODUCTION_URL string = "https://buy.itunes.apple.com/verifyReceipt"

const LAYOUT string = "2006-01-02 15:04:05"

// Request はレシート検証サーバーに送信するボディ内容です.
type Request struct {
	// ReceiptData にはBase64された検証したいレシートのデータをどうぞ.
	ReceiptData string `json:"receipt-data"`
	// Password にはapp_connect送信時のシークレットキーをどうぞ.
	Password string `json:"password"`
	// ExcludeOldTransactions にはtrueを固定値でどうぞ.
	ExcludeOldTransactions bool `json:"exclude-old-transactions"`
}

// VerifyResponse はレシート検証サーバーから返却されるボディ内容です.
type VerifyResponse struct {
	Status      int    `json:"status"`
	Environment string `json:"environment"`
	Receipt     struct {
		ReceiptType                string `json:"receipt_type"`
		AdamID                     int    `json:"adam_id"`
		AppItemID                  int    `json:"app_item_id"`
		BundleID                   string `json:"bundle_id"`
		ApplicationVersion         string `json:"application_version"`
		DownloadID                 int    `json:"download_id"`
		VersionExternalIdentifier  int    `json:"version_external_identifier"`
		ReceiptCreationDate        string `json:"receipt_creation_date"`
		ReceiptCreationDateMs      string `json:"receipt_creation_date_ms"`
		ReceiptCreationDatePst     string `json:"receipt_creation_date_pst"`
		RequestDate                string `json:"request_date"`
		RequestDateMs              string `json:"request_date_ms"`
		RequestDatePst             string `json:"request_date_pst"`
		OriginalPurchaseDate       string `json:"original_purchase_date"`
		OriginalPurchaseDateMs     string `json:"original_purchase_date_ms"`
		OriginalPurchaseDatePst    string `json:"original_purchase_date_pst"`
		OriginalApplicationVersion string `json:"original_application_version"`
		InApp                      []struct {
			Quantity                string `json:"quantity"`
			ProductID               string `json:"product_id"`
			TransactionID           string `json:"transaction_id"`
			OriginalTransactionID   string `json:"original_transaction_id"`
			PurchaseDate            string `json:"purchase_date"`
			PurchaseDateMs          string `json:"purchase_date_ms"`
			PurchaseDatePst         string `json:"purchase_date_pst"`
			OriginalPurchaseDate    string `json:"original_purchase_date"`
			OriginalPurchaseDateMs  string `json:"original_purchase_date_ms"`
			OriginalPurchaseDatePst string `json:"original_purchase_date_pst"`
			ExpiresDate             string `json:"expires_date"`
			ExpiresDateMs           string `json:"expires_date_ms"`
			ExpiresDatePst          string `json:"expires_date_pst"`
			WebOrderLineItemID      string `json:"web_order_line_item_id"`
			IsTrialPeriod           string `json:"is_trial_period"`
			IsInIntroOfferPeriod    string `json:"is_in_intro_offer_period"`
		} `json:"in_app"`
	} `json:"receipt"`
	LatestReceiptInfo []struct {
		Quantity                string `json:"quantity"`
		ProductID               string `json:"product_id"`
		TransactionID           string `json:"transaction_id"`
		OriginalTransactionID   string `json:"original_transaction_id"`
		PurchaseDate            string `json:"purchase_date"`
		PurchaseDateMs          string `json:"purchase_date_ms"`
		PurchaseDatePst         string `json:"purchase_date_pst"`
		OriginalPurchaseDate    string `json:"original_purchase_date"`
		OriginalPurchaseDateMs  string `json:"original_purchase_date_ms"`
		OriginalPurchaseDatePst string `json:"original_purchase_date_pst"`
		ExpiresDate             string `json:"expires_date"`
		ExpiresDateMs           string `json:"expires_date_ms"`
		ExpiresDatePst          string `json:"expires_date_pst"`
		WebOrderLineItemID      string `json:"web_order_line_item_id"`
		IsTrialPeriod           string `json:"is_trial_period"`
		IsInIntroOfferPeriod    string `json:"is_in_intro_offer_period"`
	} `json:"latest_receipt_info"`
	LatestReceipt      string `json:"latest_receipt"`
	PendingRenewalInfo []struct {
		ExpirationIntent       string `json:"expiration_intent"`
		AutoRenewProductID     string `json:"auto_renew_product_id"`
		OriginalTransactionID  string `json:"original_transaction_id"`
		IsInBillingRetryPeriod string `json:"is_in_billing_retry_period"`
		ProductID              string `json:"product_id"`
		AutoRenewStatus        string `json:"auto_renew_status"`
	} `json:"pending_renewal_info"`
}

// VerifyResult はレシートの検証結果です.
type VerifyResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RecieptPerUid はUIDごとのレシート管理情報です.
type RecieptPerUid struct {
	Uid           string `json:"uid"`
	TransactionId string `json:"transaction_id"`
}

// VerifyReceipt はレシートの検証を実施します.
func VerifyReceipt(uid string, data string) *bytes.Buffer {

	// 環境変数読み込み.
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil
	}

	// Firebase初期化.
	ctx := context.Background()
	opt := option.WithCredentialsFile("./motion-dev-d0877-firebase-adminsdk-sn5vm-90d4884363.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	defer client.Close()

	// リクエストボディ作成.
	reqBody := new(Request)
	reqBody.ReceiptData = data
	reqBody.Password = os.Getenv("PASSWORD")
	reqBody.ExcludeOldTransactions = true

	// リクエストボディをJSON変換.
	request_json, _ := json.Marshal(reqBody)

	// AppConnectへPOSTリクエスト.
	res, err := http.Post(SANDBOX_URL, "application/json", bytes.NewBuffer(request_json))
	if err != nil {
		log.Fatal("Post request error occured.")
	}
	defer res.Body.Close()

	// レスポンス内容を構造体に変換
	verify_response := new(VerifyResponse)
	buf := new(bytes.Buffer)
	io.Copy(buf, res.Body)
	json.Unmarshal(buf.Bytes(), verify_response)

	// 返却されたレスポンスのステータスによって処理を分岐します.
	switch verify_response.Status {

	case 0: // 検証成功.

		// レシートが自身のアプリのものかを検証
		if verify_response.Receipt.BundleID == os.Getenv("BUNDLE_ID") {

			// 戻り値のレシートから最新のものを抽出.
			var latest_transaction_id string
			var latest_expires_date string
			for _, v := range verify_response.Receipt.InApp {
				if latest_transaction_id < v.TransactionID {
					latest_transaction_id = v.TransactionID
					latest_expires_date = v.ExpiresDate
				}
			}

			// DBからトランザクションIDを取得.
			field, err := client.Collection("Receipt").Doc(uid).Get(ctx)
			if err != nil {
				log.Fatal("Faild to get transactionId")
				return nil
			}

			// 取得したReceiptコレクションのドキュメント.
			data := field.Data()

			// 登録済みの場合、レシート検証は失敗.
			if data != nil && data["transactionId"] == latest_transaction_id {
				log.Fatal("This receipt is already registed.")
				return nil
			}

			// 未登録の場合、DBに登録してレシート検証成功.
			_, err = client.Collection("Receipt").Doc(uid).Set(ctx, map[string]interface{}{
				"uid":           uid,
				"transactionId": latest_transaction_id,
			})
			if err != nil {
				log.Fatal("Failed to Add TransactionID.")
				return nil
			}

			// 有効期限の検証.
			now := time.Now()
			if now.Before(stringToTime(latest_expires_date)) {
				log.Fatal("This receipt is out of date.")
			}

			// 検証結果を作成.
			verify_result := new(VerifyResult)
			verify_result.Code = 200
			verify_result.Message = ""

			// 検証結果をJSON変換.
			json_verify_result, _ := json.Marshal(verify_result)

			// アプリ側へレスポンス.
			return bytes.NewBuffer(json_verify_result)
		}

	case 21007: // SandBox環境に初回リクエストした際のステータスコードのためリトライする.
		// AppConnectへPOSTリクエスト.
		res_retry, err_retry := http.Post(SANDBOX_URL, "application/json", bytes.NewBuffer(request_json))
		if err_retry != nil {
			log.Fatal("Post request error occured.")
		}
		defer res_retry.Body.Close()

		// レスポンス内容を構造体に変換
		verify_response_retry := new(VerifyResponse)
		buf_retry := new(bytes.Buffer)
		io.Copy(buf_retry, res_retry.Body)
		json.Unmarshal(buf_retry.Bytes(), verify_response_retry)

		// 返却されたレスポンスのステータスによって処理を分岐します.
		switch verify_response_retry.Status {

		case 0: // 検証成功

			if verify_response_retry.Receipt.BundleID == os.Getenv("BUNDLE_ID") {

				// 戻り値のレシートから最新のものを抽出.

				// DB登録済みか検証

				// 登録済みの場合、レシート検証は失敗

				// 未登録の場合、DBに登録してレシート検証成功

				// 検証結果を作成.
				verify_result_retry := new(VerifyResult)
				verify_result_retry.Code = 200
				verify_result_retry.Message = ""

				// 検証結果をJSON変換.
				json_verify_result_retry, _ := json.Marshal(verify_result_retry)

				// アプリ側へレスポンス.
				return bytes.NewBuffer(json_verify_result_retry)
			}
		}
	}

	// 失敗時の検証結果を作成.
	verify_result := new(VerifyResult)
	verify_result.Code = 400
	verify_result.Message = "Faild to verify reciept."

	// 検証結果をJSON変換.
	json_verify_result, _ := json.Marshal(verify_result)

	// アプリ側へレスポンス.
	return bytes.NewBuffer(json_verify_result)
}

func stringToTime(str string) time.Time {
	t, _ := time.Parse(LAYOUT, str)
	return t
}
