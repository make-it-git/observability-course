package com.example.demo;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;

@RestController
public class HelloController {

    private final RestTemplate restTemplate = new RestTemplate();

    @GetMapping("/")
    public String hello() {
        return "Hello, Spring boot!";
    }

    @GetMapping("/httpbin/headers")
    public String httpbinHeaders() {
        String url = "https://httpbin.org/headers";
        try {
            ResponseEntity<String> response = restTemplate.getForEntity(url, String.class);
            return "Response from httpbin.org/headers: " + response.getBody();
        } catch (Exception e) {
            return "Error fetching data from httpbin.org/headers: " + e.getMessage();
        }
    }
}