package com.financials.api;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class IndexController {

	@RequestMapping(value = "/", method = RequestMethod.POST)
    public ResponseEntity<Retirement> index(@RequestBody Plan plan) {
		
		Retirement retirement = new Retirement(plan);
		retirement.calculate();
		
        return new ResponseEntity<Retirement>(retirement, HttpStatus.OK);
    }
}
