package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Check if we have the right number of arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: <program> <filename1> <filename2>")
		os.Exit(1)
	}

	// Extract filenames from command line arguments
	filenameA := os.Args[1]
	filenameB := os.Args[2]

	fmt.Printf("Processing files: %s and %s\n", filenameA, filenameB)

	compare(filenameA, filenameB)
}

func compare(filenameA, filenameB string) {
	fmt.Printf("A is %q and B is %q\n", filenameA, filenameB)

	k6Req, k6Resp, err := readCDPFile(filenameA)
	if err != nil {
		fmt.Printf("\nError reading file A:", err)
		return
	}

	k6UnmatchedReq, k6UnmatchedResp, k6NoIDResponses := linkRequestResponse(k6Req, k6Resp)
	fmt.Printf("\nA requests with no matching response:\n\n")
	for _, r := range k6UnmatchedReq {
		fmt.Println(r)
	}
	fmt.Printf("\nA responses with no matching request:\n\n")
	for _, r := range k6UnmatchedResp {
		fmt.Println(r)
	}
	fmt.Printf("\nThere are %d No ID A responses\n", len(k6NoIDResponses))

	pwReq, pwResp, err := readCDPFile(filenameB)
	if err != nil {
		fmt.Println("Error reading B file:", err)
		return
	}

	pwUnmatchedReq, pwUnmatchedResp, pwNoIDResponses := linkRequestResponse(pwReq, pwResp)
	fmt.Printf("\nB requests with no matching response:\n\n")
	for _, r := range pwUnmatchedReq {
		fmt.Println(r)
	}
	fmt.Printf("\nB responses with no matching request:\n\n")
	for _, r := range pwUnmatchedResp {
		fmt.Println(r)
	}
	fmt.Printf("There are %d No ID B responses\n", len(pwNoIDResponses))

	ink6NotInPw := filterRequests(k6Req, pwReq)
	fmt.Printf("\nRequests in A but not in B:\n")
	for _, r := range ink6NotInPw {
		fmt.Println(r)
	}

	inPwNotInK6 := filterRequests(pwReq, k6Req)
	fmt.Printf("\nRequests in B but not in A:\n")
	for _, r := range inPwNotInK6 {
		fmt.Println(r)
	}

	respInk6NotInPw := filterResponses(k6Resp, pwResp)
	fmt.Printf("\nResponses in A but not in B:\n")
	for _, r := range respInk6NotInPw {
		fmt.Println(r)
	}

	respInPwNotInK6 := filterResponses(pwResp, k6Resp)
	fmt.Printf("\nResponses in B but not in A:\n")
	for _, r := range respInPwNotInK6 {
		fmt.Println(r)
	}
}

func filterResponses(sliceA, sliceB []cdpResponse) []cdpResponse {
	// Create a map to store methods from sliceB
	methodsInB := make(map[string]bool)

	// Populate the map with methods from sliceB
	for _, resp := range sliceB {
		if method, found := getMethodFromResult(resp); found {
			methodsInB[method] = true
		}
	}

	// Create a slice to hold responses that are in A but not in B
	var uniqueInA []cdpResponse

	// Check each response in sliceA to see if its method is not in sliceB
	for _, resp := range sliceA {
		if method, found := getMethodFromResult(resp); found {
			if _, inB := methodsInB[method]; !inB {
				uniqueInA = append(uniqueInA, resp)
			}
		}
	}

	return uniqueInA
}

// getMethodFromResult extracts the "method" value from the result map if it exists
func getMethodFromResult(resp cdpResponse) (string, bool) {
	if resp.msg.Method != nil {
		return *resp.msg.Method, true
	}
	return "", false
}

func filterRequests(sliceA, sliceB []cdpRequest) []cdpRequest {
	// Create a map to store the methods from sliceB
	methodsInB := make(map[string]bool)

	// Populate the map with methods from sliceB
	for _, req := range sliceB {
		if req.msg != nil {
			methodsInB[req.msg.Method] = true
		}
	}

	// Create a slice to hold requests that are in A but not in B
	var uniqueInA []cdpRequest

	// Check each request in sliceA to see if its method is not in sliceB
	for _, req := range sliceA {
		if req.msg != nil {
			// If the method from A is not found in B, add to result
			if _, found := methodsInB[req.msg.Method]; !found {
				uniqueInA = append(uniqueInA, req)
			}
		}
	}

	return uniqueInA
}

func linkRequestResponse(requests []cdpRequest, responses []cdpResponse) ([]cdpRequest, []cdpResponse, []cdpResponse) {
	// Create a map to quickly find responses by ID and track matched responses
	responseMap := make(map[int]*cdpResponse)
	matchedResponses := make(map[int]bool)
	var noIDResponses []cdpResponse // Slice to hold responses with no valid ID

	// Populate the map with responses that have a valid ID
	for i, resp := range responses {
		if resp.msg != nil && resp.msg.ID != nil && *resp.msg.ID != 0 {
			responseMap[*resp.msg.ID] = &responses[i]
		} else {
			noIDResponses = append(noIDResponses, resp)
		}
	}

	// Prepare slices to hold unmatched requests and responses
	var unmatchedRequests []cdpRequest
	var unmatchedResponses []cdpResponse

	// Link each request to its corresponding response
	for i, req := range requests {
		if req.msg != nil && req.msg.ID != 0 {
			if resp, found := responseMap[req.msg.ID]; found {
				requests[i].response = resp
				matchedResponses[req.msg.ID] = true
			} else {
				unmatchedRequests = append(unmatchedRequests, req)
			}
		} else {
			unmatchedRequests = append(unmatchedRequests, req)
		}
	}

	// Identify unmatched responses
	for id, resp := range responseMap {
		if !matchedResponses[id] {
			unmatchedResponses = append(unmatchedResponses, *resp)
		}
	}

	return unmatchedRequests, unmatchedResponses, noIDResponses
}

type cdpRequestMsg struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
}

type cdpRequest struct {
	raw      string
	msg      *cdpRequestMsg
	response *cdpResponse
}

type cdpResponseMsg struct {
	ID     *int                   `json:"id,omitempty"`
	Method *string                `json:"method,omitempty"`
	Result map[string]interface{} `json:"result"`
}

type cdpResponse struct {
	raw string
	msg *cdpResponseMsg
}

func readCDPFile(filename string) ([]cdpRequest, []cdpResponse, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var requests []cdpRequest
	var responses []cdpResponse
	scanner := bufio.NewScanner(file)

	// Increase the buffer size for the scanner
	const maxCapacity = 512 * 1024 // Increase buffer size to 512KB, adjust this as needed
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "->") {
			// Remove the "->" prefix and any leading whitespace
			jsonStr := strings.TrimSpace(line[2:])
			var msg cdpRequestMsg
			if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
				fmt.Println(jsonStr)
				return nil, nil, fmt.Errorf("unmarshaling request: %w", err)
			}
			requests = append(requests, cdpRequest{raw: line, msg: &msg})
		} else if strings.HasPrefix(line, "<-") {
			jsonStr := strings.TrimSpace(line[2:])
			var msg cdpResponseMsg
			if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
				fmt.Println(jsonStr)
				return nil, nil, fmt.Errorf("unmarshaling response: %w", err)
			}
			responses = append(responses, cdpResponse{raw: line, msg: &msg})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return requests, responses, nil
}
