FROM openjdk:17-jdk-alpine

WORKDIR /app

COPY Main.java /app/Main.java

RUN javac Main.java

CMD ["java", "-Xms80m", "Main"]
