# 울릉 래퍼
울릉 래퍼는 Go로 작성된 울릉 jar 파일 실행용 프로그램입니다.  
## 샘플 명령어
```
ulleungw sample.uln -t --use-version 2020.08.20.01 --dok system.dok

sample.uln -> 코드 파일 이름

-t -> ulleungt를 사용해 .java로 변환
-c -> ulleungc를 사용해 .class로 변환 (미구현)

--verison -> 사용할 .jar 버전 이름 (생략할 경우 가장 최신의 안정 버전으로 실행)

--dok -> 사용할 .dok 파일 선언 (다중 가능, 생략 가능)
system.dok -> .dok 파일 이름