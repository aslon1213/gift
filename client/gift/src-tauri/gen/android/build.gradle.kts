signingConfigs {
    create("release") {
        storeFile = file("../../../../../.keystore/gift.keystore")
        storePassword = System.getenv("KEYSTORE_PASSWORD")
        keyAlias = "gift"
        keyPassword = System.getenv("KEY_PASSWORD")
    }
}   

buildTypes {
    release {
        signingConfig = signingConfigs.getByName("release")
    }
}


buildscript {
    repositories {
        google()
        mavenCentral()
    }
    dependencies {
        classpath("com.android.tools.build:gradle:8.11.0")
        classpath("org.jetbrains.kotlin:kotlin-gradle-plugin:1.9.25")
    }
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

tasks.register("clean").configure {
    delete("build")
}

